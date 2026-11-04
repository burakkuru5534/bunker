import 'dart:async';
import 'dart:convert';

import 'package:siginak/core/realtime/ws_socket.dart';

/// Test için sahte WebSocket.
class FakeWsSocket implements WsSocket {
  final _incoming = StreamController<Object?>();
  final List<String> sent = [];
  int? _closeCode;
  bool closedByClient = false;

  @override
  Stream<Object?> get messages => _incoming.stream;

  @override
  void send(String data) => sent.add(data);

  @override
  Future<void> close([int? code, String? reason]) async {
    closedByClient = true;
    _closeCode ??= code;
    if (!_incoming.isClosed) await _incoming.close();
  }

  @override
  int? get closeCode => _closeCode;

  /// Sunucudan v1 zarfı gönderir.
  void serverSend(String type, {int? stateVersion, Object? payload}) {
    _incoming.add(
      jsonEncode({
        'v': 1,
        'type': type,
        'room_id': 'r',
        'state_version': ?stateVersion,
        'ts': '2026-11-02T18:00:00Z',
        'payload': payload ?? <String, Object?>{},
      }),
    );
  }

  void serverSendRaw(Object? data) => _incoming.add(data);

  /// Sunucu bağlantıyı belirtilen kodla kapatır.
  void serverClose(int code) {
    _closeCode = code;
    unawaited(_incoming.close());
  }
}

/// Bağlantı isteklerini kaydeden sahte bağlayıcı.
class FakeWsConnector {
  final List<Uri> uris = [];
  final List<Map<String, String>> headers = [];
  final List<FakeWsSocket> sockets = [];

  /// `true` ise bir sonraki bağlantı denemesi başarısız olur.
  int failNext = 0;

  Future<WsSocket> call(Uri uri, Map<String, String> h) async {
    uris.add(uri);
    headers.add(h);
    if (failNext > 0) {
      failNext--;
      throw Exception('bağlanılamadı');
    }
    final socket = FakeWsSocket();
    sockets.add(socket);
    return socket;
  }

  FakeWsSocket get last => sockets.last;
}

/// Lobi üyesi (docs/api.md RoomView.members).
Map<String, Object?> wsMember(
  String userId,
  String nickname, {
  bool ready = false,
  String role = 'player',
}) => {
  'user_id': userId,
  'nickname': nickname,
  'role': role,
  'is_ready': ready,
  'connected': true,
};

/// `session.snapshot` yükü (docs/api.md §5).
Map<String, Object?> snapshotPayload({
  required String roomId,
  required String me,
  required List<Map<String, Object?>> members,
  String? host,
  String status = 'lobby',
  int capacity = 2,
  int version = 1,
  String inviteCode = 'K7M2QX',
  Map<String, Object?>? game,
}) => {
  'server_time': '2026-11-04T10:00:00Z',
  'me': {'user_id': me, 'role': 'player'},
  'room': {
    'id': roomId,
    'invite_code': inviteCode,
    'status': status,
    'host_user_id': host ?? members.first['user_id'],
    'capacity': capacity,
    'current_game_id': null,
    'version': version,
    'members': members,
  },
  'game': game,
  'chat': <Object?>[],
};

/// Test senaryosu (`scenario` nesnesi).
const testScenario = {
  'id': 'meteor',
  'title': 'Göktaşı',
  'disaster': 'Dev bir göktaşı okyanusa düştü. Toz güneşi kapatacak.',
  'duration': '2 yıl',
  'bunker': ['Su arıtma çalışıyor', 'Yakıt sınırlı', 'Teknik kütüphane'],
  'outside': 'İklim kararsız; asit yağmurları sürüyor.',
};

/// `game.started` yükü: koltuk sırasıyla oyuncular.
Map<String, Object?> gameStartedPayload(
  List<(String, String)> players, {
  int capacity = 2,
  int totalRounds = 2,
}) => {
  'game_id': 'g1',
  'scenario': testScenario,
  'capacity': capacity,
  'total_rounds': totalRounds,
  'participants': [
    for (final (i, (id, nick)) in players.indexed)
      {'user_id': id, 'nickname': nick, 'seat': i},
  ],
  'reveal_order': [for (final (id, _) in players) id],
};

/// `game.your_cards` / `my_cards` kartları.
List<Map<String, Object?>> myCardsJson({Set<String> revealed = const {}}) => [
  for (final (c, t) in [
    ('profession', 'Acil tıp doktoru'),
    ('health', 'Astım'),
    ('hobby', 'Arıcılık'),
    ('item', 'Termos'),
    ('secret', 'Gizli bir günlük tutuyor'),
  ])
    {'category': c, 'text': t, 'revealed': revealed.contains(c)},
];

/// `session.snapshot.game` (sürmekte olan oyun).
Map<String, Object?> snapshotGame({
  required List<Map<String, Object?>> participants,
  String phase = 'REVEAL',
  int round = 1,
  int totalRounds = 2,
  int stateVersion = 10,
  String? turnUserId,
  String? deadline,
  List<Map<String, Object?>>? myCards,
  Map<String, Object?>? myActionState,
  int readyCount = 0,
  Map<String, Object?>? voteProgress,
  Map<String, Object?>? tiebreak,
  Map<String, Object?>? lastRoundResult,
}) => {
  'id': 'g1',
  'state_version': stateVersion,
  'phase': phase,
  'round': round,
  'total_rounds': totalRounds,
  'capacity': 2,
  'phase_deadline_at': deadline,
  'scenario': testScenario,
  'turn_user_id': turnUserId,
  'participants': participants,
  'my_cards': myCards ?? myCardsJson(),
  'my_action_state':
      myActionState ??
      {'voted': false, 'discussion_ready': false, 'tiebreak_voted': false},
  'discussion_ready_count': readyCount,
  'vote_progress': voteProgress ?? {'cast': 0, 'total': 0},
  'tiebreak': tiebreak,
  'last_round_result': lastRoundResult,
};

/// Snapshot katılımcısı.
Map<String, Object?> participantJson(
  String id,
  String nick,
  int seat, {
  String status = 'alive',
  bool connected = true,
  Map<String, String> revealed = const {},
}) => {
  'user_id': id,
  'nickname': nick,
  'seat': seat,
  'status': status,
  'connected': connected,
  'revealed_cards': [
    for (final e in revealed.entries) {'category': e.key, 'text': e.value},
  ],
};

/// `game.finished` / `GET result` yükü: u-guest ve u2 kazanır, u3 elenir.
Map<String, Object?> resultJson({String endReason = 'capacity_reached'}) => {
  'game_id': 'g1',
  'finished_at': '2026-11-04T10:30:00Z',
  'end_reason': endReason,
  'scenario': {'id': 'meteor', 'title': 'Göktaşı'},
  'capacity': 2,
  'players': [
    {
      'user_id': 'u-guest',
      'nickname': 'Ayşe',
      'is_winner': true,
      'status': 'alive',
      'eliminated_round': null,
      'cards': [
        {'category': 'secret', 'text': 'Gizli günlük'},
        {'category': 'profession', 'text': 'Acil tıp doktoru'},
      ],
    },
    {
      'user_id': 'u2',
      'nickname': 'Mehmet',
      'is_winner': true,
      'status': 'alive',
      'eliminated_round': null,
      'cards': [
        {'category': 'profession', 'text': 'Aşçı'},
      ],
    },
    {
      'user_id': 'u3',
      'nickname': 'Zeynep',
      'is_winner': false,
      'status': 'eliminated',
      'eliminated_round': 1,
      'cards': [
        {'category': 'profession', 'text': 'Astrolog'},
      ],
    },
  ],
  'rounds': [
    {
      'round': 1,
      'eliminated_user_id': 'u3',
      'method': 'vote',
      'tally': [
        {'user_id': 'u3', 'votes': 2},
        {'user_id': 'u2', 'votes': 1},
      ],
    },
  ],
};
