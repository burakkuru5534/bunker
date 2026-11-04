import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/realtime/realtime_client.dart';
import 'package:siginak/core/realtime/realtime_client_provider.dart';
import 'package:siginak/core/realtime/ws_envelope.dart';
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
  });

  /// İlk `session.snapshot` gelene kadar `null`.
  final RoomView? room;
  final String? myUserId;
  final RealtimeState connection;

  /// Süren (veya biten) oyun; lobide `null`.
  final GameView? game;
  final RoomExit? exit;

  RoomMember? get me => room?.memberOf(myUserId);
  bool get isHost => room != null && room!.hostUserId == myUserId;

  RoomState copyWith({
    RoomView? room,
    String? myUserId,
    RealtimeState? connection,
    GameView? Function()? game,
    RoomExit? exit,
  }) => RoomState(
    room: room ?? this.room,
    myUserId: myUserId ?? this.myUserId,
    connection: connection ?? this.connection,
    game: game != null ? game() : this.game,
    exit: exit ?? this.exit,
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
      state = state.copyWith(
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

    if (e.type.startsWith('game.') && e.type != 'game.started') {
      final game = state.game;
      if (game == null) return;
      final next = game.apply(
        e.type,
        p,
        receivedAt: now,
        myUserId: state.myUserId,
      );
      state = state.copyWith(
        game: () => next,
        room: e.type == 'game.finished'
            ? room.copyWith(status: RoomStatus.finished)
            : null,
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

  static RoomView _without(RoomView room, Object? userId) => room.copyWith(
    members: [
      for (final m in room.members)
        if (m.userId != userId) m,
    ],
  );
}

final roomControllerProvider = NotifierProvider.autoDispose
    .family<RoomController, RoomState, String>(RoomController.new);
