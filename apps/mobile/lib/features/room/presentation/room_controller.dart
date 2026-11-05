import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/realtime/realtime_client.dart';
import 'package:siginak/core/realtime/realtime_client_provider.dart';
import 'package:siginak/core/realtime/ws_envelope.dart';
import 'package:siginak/core/util/uuid.dart';
import 'package:siginak/features/chat/chat.dart';
import 'package:siginak/features/game/game.dart';
import 'package:siginak/features/room/data/room_models.dart';
import 'package:siginak/features/room/data/room_repository.dart';
import 'package:siginak/features/session/session.dart';

/// Oyun kuralları: oyuncu sayısı sınırları (docs/game-design.md §1).
const minPlayers = 4;
const maxPlayers = 10;

/// Kullanıcının odadan çıkış nedeni (ekran ana ekrana döner).
enum RoomExit {
  /// Kendi isteğiyle ayrıldı.
  left,

  /// Oda sahibi tarafından atıldı (`room.member_kicked`).
  kicked,

  /// Sunucu üyeliği reddetti (WS 4003; ör. oda kapandı).
  forbidden,
}

/// Oda ekranının durumu: sunucudan gelen oda görünümü + bağlantı.
@immutable
class RoomState {
  const RoomState({
    required this.connection,
    this.room,
    this.myUserId,
    this.game,
    this.exit,
    this.result,
    this.chat = const ChatState(),
  });

  /// İlk `session.snapshot` gelene kadar `null`.
  final RoomView? room;
  final String? myUserId;
  final RealtimeState connection;

  /// Süren (veya biten) oyun; lobide `null`.
  final GameView? game;
  final RoomExit? exit;

  /// `game.finished` ile gelen sonuç (kaçırıldıysa ekran sunucudan ister).
  final GameResult? result;

  /// Sohbet (snapshot geçmişi + `chat.message` + bekleyen mesajlarım).
  final ChatState chat;

  RoomMember? get me => room?.memberOf(myUserId);

  /// Yazabilir miyim? Lobide ve oyun bitince her üye; oyun sürerken yalnızca
  /// hayattaki oyuncular (docs/api.md "Sohbet"; karar sunucuda).
  bool get canChat {
    final r = room;
    if (r == null || me == null) return false;
    if (r.status != RoomStatus.inGame) return true;
    return game?.participantOf(myUserId)?.isAlive ?? false;
  }

  bool get isHost => room != null && room!.hostUserId == myUserId;

  RoomState copyWith({
    RoomView? room,
    String? myUserId,
    RealtimeState? connection,
    GameView? Function()? game,
    RoomExit? exit,
    GameResult? Function()? result,
    ChatState? chat,
  }) => RoomState(
    room: room ?? this.room,
    myUserId: myUserId ?? this.myUserId,
    connection: connection ?? this.connection,
    game: game != null ? game() : this.game,
    exit: exit ?? this.exit,
    result: result != null ? result() : this.result,
    chat: chat ?? this.chat,
  );
}

/// Başlatma neden yapılamıyor (istemci tarafı ön kontrol; karar sunucuda).
sealed class StartBlocker {
  const StartBlocker();
}

class TooFewPlayers extends StartBlocker {
  const TooFewPlayers(this.count);
  final int count;
}

class TooManyPlayers extends StartBlocker {
  const TooManyPlayers();
}

class NotAllReady extends StartBlocker {
  const NotAllReady(this.ready, this.total);
  final int ready;
  final int total;
}

/// Lobide başlatmayı engelleyen durum; başlatılabiliyorsa `null`.
StartBlocker? startBlockerOf(RoomView room) {
  final players = room.players.toList();
  if (players.length < minPlayers) return TooFewPlayers(players.length);
  if (players.length > maxPlayers) return const TooManyPlayers();
  final ready = players.where((p) => p.isReady).length;
  if (ready < players.length) return NotAllReady(ready, players.length);
  return null;
}

/// Bir odanın gerçek zamanlı durumu (docs/ui/screen-map.md §3.4).
///
/// Ekran açılınca odaya WS ile bağlanır, kapanınca bağlantıyı kapatır.
/// Durum yalnızca sunucudan gelir: `session.snapshot` tam görüntüyü,
/// `room.*` / `game.*` olayları değişiklikleri taşır. Eylemler REST ile
/// gider; sonuçları olaylarla yansır.
class RoomController extends Notifier<RoomState> {
  RoomController(this.roomId);

  final String roomId;

  RealtimeClient get _client => ref.read(realtimeClientProvider);
  RoomRepository get _repo => ref.read(roomRepositoryProvider);

  @override
  RoomState build() {
    final client = _client;
    final events = client.events.listen(_onEvent);
    final states = client.states.listen(_onConnection);
    ref.onDispose(() {
      unawaited(events.cancel());
      unawaited(states.cancel());
      unawaited(client.disconnect());
    });
    client.connect(roomId);
    return RoomState(
      connection: client.state,
      myUserId: ref.read(sessionControllerProvider).userId,
    );
  }

  /// Uygulama ön plana geldi: bağlıysa eşitle, değilse hemen bağlan.
  void onAppResumed() => _client.onAppResumed();

  /// Başka cihazda açıldığı için duran bağlantıyı yeniden kurar.
  void reconnect() => _client.connect(roomId);

  Future<void> setReady({required bool ready}) =>
      _repo.setReady(roomId, ready: ready);

  Future<void> setCapacity(int capacity) => _repo.setCapacity(roomId, capacity);

  Future<void> kick(String userId) => _repo.kick(roomId, userId);

  Future<void> start() => _repo.start(roomId);

  /// Oyun eylemi (kart aç, hazırım, oy). Başarılıysa yalnızca benim
  /// bilebileceğim durum (oy verdim / hazırım) işaretlenir; gerisi olaylarla.
  Future<void> act(GameAction action) async {
    final game = state.game;
    if (game == null) return;
    try {
      await ref.read(gameRepositoryProvider).act(game.id, game.round, action);
    } on ApiException catch (e) {
      // Oy zaten verilmiş (ör. başka cihazdan): işareti düzelt, hatayı ilet.
      if (e.code == 'ALREADY_VOTED') _markDone(game.id, action);
      rethrow;
    }
    _markDone(game.id, action);
  }

  void _markDone(String gameId, GameAction action) {
    final current = state.game;
    if (!ref.mounted || current == null || current.id != gameId) return;
    state = state.copyWith(
      game: () => switch (action) {
        Vote() => current.withMyAction(voted: true),
        DiscussionReady() => current.withMyAction(discussionReady: true),
        TiebreakVote() => current.withMyAction(tiebreakVoted: true),
        RevealCard() => current,
      },
    );
  }

  /// Sohbet mesajı gönderir. Önce "gönderiliyor" olarak görünür; başarıda
  /// sunucu mesajı (WS ile de gelir, kimliğe göre tek kalır), hatada
  /// "gönderilemedi" (dokununca aynı kimlikle yeniden denenir).
  Future<void> sendChat(String text) async {
    final pending = PendingMessage(clientMsgId: uuidV4(), text: text);
    state = state.copyWith(
      chat: state.chat.withPending([...state.chat.pending, pending]),
    );
    await _deliver(pending);
  }

  /// Gönderilemeyen mesajı aynı `client_msg_id` ile yeniden dener.
  Future<void> retryChat(PendingMessage failed) async {
    final again = failed.retrying();
    state = state.copyWith(
      chat: state.chat.withPending([
        for (final p in state.chat.pending)
          p.clientMsgId == failed.clientMsgId ? again : p,
      ]),
    );
    await _deliver(again);
  }

  Future<void> _deliver(PendingMessage p) async {
    try {
      final sent = await ref
          .read(chatRepositoryProvider)
          .send(roomId, clientMsgId: p.clientMsgId, text: p.text);
      if (!ref.mounted) return;
      var chat = state.chat.withPending([
        for (final x in state.chat.pending)
          if (x.clientMsgId != p.clientMsgId) x,
      ]);
      // WS kopuksa yankı gelmez: kendi mesajımı ekle. Yankı zaten geldiyse
      // (sunucunun maskelediği metinle) dokunma.
      if (!chat.messages.any((m) => m.id == sent.id)) {
        chat = chat.withMessage(
          ChatMessage(
            id: sent.id,
            userId: state.myUserId ?? '',
            nickname: state.me?.nickname ?? '',
            text: p.text,
            createdAt: sent.createdAt,
          ),
        );
      }
      state = state.copyWith(chat: chat);
    } on ApiException catch (e) {
      if (!ref.mounted) return;
      state = state.copyWith(
        chat: state.chat.withPending([
          for (final x in state.chat.pending)
            x.clientMsgId == p.clientMsgId ? x.failedWith(e) : x,
        ]),
      );
    }
  }

  /// Engellediğim kullanıcının mevcut mesajlarını gizler.
  void hideChatFrom(String userId) {
    state = state.copyWith(chat: state.chat.withoutUser(userId));
  }

  /// Sohbet paneli açıkken: mesajlar görüldü.
  void markChatSeen() {
    final seen = state.chat.seen();
    if (seen.lastSeenId != state.chat.lastSeenId) {
      state = state.copyWith(chat: seen);
    }
  }

  /// Tekrar oyna (oda sahibi): yanıt yeni lobi görünümüdür.
  Future<void> rematch() async {
    final room = await _repo.rematch(roomId);
    if (!ref.mounted) return;
    state = state.copyWith(room: room, game: () => null, result: () => null);
  }

  /// Odadan ayrılır (idempotent); başarılıysa çıkış `left` olur.
  Future<void> leave() async {
    await _repo.leave(roomId);
    if (!ref.mounted) return;
    state = state.copyWith(exit: RoomExit.left);
    unawaited(_client.disconnect());
  }

  void _onConnection(RealtimeState connection) {
    if (!ref.mounted) return;
    var next = state.copyWith(connection: connection);
    if (connection.status == RealtimeStatus.stopped &&
        connection.stopReason == RealtimeStopReason.forbidden &&
        state.exit == null) {
      next = next.copyWith(exit: RoomExit.forbidden);
    }
    state = next;
  }

  void _onEvent(WsEnvelope e) {
    if (!ref.mounted || state.exit != null) return;
    try {
      _apply(e);
    } on Exception {
      // ApiException / FormatException: beklenmeyen biçim → tam görüntü.
      _client.requestSync();
    }
  }

  void _apply(WsEnvelope e) {
    final p = e.payload;
    final room = state.room;
    final now = DateTime.now();

    if (e.type == 'session.snapshot') {
      final me = p['me'];
      final game = p['game'];
      final serverTime = p['server_time'];
      final history = p['chat'];
      state = state.copyWith(
        chat: state.chat.withHistory([
          if (history is List)
            for (final m in history) ?ChatMessage.tryParse(m),
        ]),
        room: RoomView.fromJson(p['room']),
        myUserId: me is Map<String, Object?> && me['user_id'] is String
            ? me['user_id']! as String
            : null,
        game: () => game == null
            ? null
            : GameView.fromSnapshot(
                game,
                serverTime: serverTime is String
                    ? DateTime.tryParse(serverTime)
                    : null,
                receivedAt: now,
              ),
      );
      return;
    }
    if (room == null) return; // snapshot'tan önce olay gelmez (istemci süzer)

    if (e.type == 'chat.message') {
      final msg = ChatMessage.tryParse(p);
      if (msg != null) {
        state = state.copyWith(chat: state.chat.withMessage(msg));
      }
      return;
    }

    if (e.type.startsWith('game.') && e.type != 'game.started') {
      final game = state.game;
      if (game == null) return;
      final next = game.apply(
        e.type,
        p,
        receivedAt: now,
        myUserId: state.myUserId,
      );
      final finished = e.type == 'game.finished';
      state = state.copyWith(
        game: () => next,
        room: finished ? room.copyWith(status: RoomStatus.finished) : null,
        result: finished ? () => _tryResult(p) : null,
      );
      return;
    }

    final version = e.stateVersion;
    RoomView? next;
    switch (e.type) {
      case 'room.member_joined':
        final member = RoomMember.fromJson(p['member']);
        next = room.copyWith(
          members: [
            for (final m in room.members)
              if (m.userId != member.userId) m,
            member,
          ],
        );
      case 'room.member_left':
        next = _without(room, p['user_id']);
      case 'room.member_kicked':
        if (p['user_id'] == state.myUserId) {
          state = state.copyWith(exit: RoomExit.kicked);
          return;
        }
        next = _without(room, p['user_id']);
      case 'room.ready_changed':
        final ready = p['is_ready'];
        if (ready is! bool) return;
        next = room.copyWith(
          members: [
            for (final m in room.members)
              m.userId == p['user_id'] ? m.copyWith(isReady: ready) : m,
          ],
        );
      case 'room.settings_changed':
        final capacity = p['capacity'];
        if (capacity is! int) return;
        next = room.copyWith(capacity: capacity);
      case 'room.host_changed':
        final host = p['host_user_id'];
        if (host is! String) return;
        next = room.copyWith(hostUserId: host);
      case 'room.returned_to_lobby':
        state = state.copyWith(
          room: RoomView.fromJson(p['room']),
          game: () => null,
          result: () => null,
        );
        return;
      case 'game.started':
        final gameId = p['game_id'];
        state = state.copyWith(
          room: room.copyWith(
            status: RoomStatus.inGame,
            currentGameId: () => gameId is String ? gameId : null,
          ),
          game: () => GameView.fromStarted(p, receivedAt: now),
        );
        return;
      default:
        return;
    }
    final isRoomEvent = e.type.startsWith('room.');
    state = state.copyWith(
      room: isRoomEvent && version != null
          ? next.copyWith(version: version)
          : next,
    );
  }

  static GameResult? _tryResult(Map<String, Object?> p) {
    try {
      return GameResult.fromJson(p);
    } on FormatException {
      return null; // ekran sonucu sunucudan ister
    }
  }

  static RoomView _without(RoomView room, Object? userId) => room.copyWith(
    members: [
      for (final m in room.members)
        if (m.userId != userId) m,
    ],
  );
}

final roomControllerProvider = NotifierProvider.autoDispose
    .family<RoomController, RoomState, String>(RoomController.new);
