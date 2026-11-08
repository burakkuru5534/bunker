import 'package:flutter/foundation.dart';

import 'package:siginak/core/network/api_exception.dart';

/// Odanın durumu (docs/api.md "Oda görünümü").
enum RoomStatus { lobby, inGame, finished }

/// Üyenin odadaki rolü.
enum MemberRole { player, spectator }

@immutable
class RoomMember {
  const RoomMember({
    required this.userId,
    required this.nickname,
    required this.role,
    required this.isReady,
    required this.connected,
  });

  final String userId;
  final String nickname;
  final MemberRole role;
  final bool isReady;
  final bool connected;

  RoomMember copyWith({bool? isReady, MemberRole? role}) => RoomMember(
    userId: userId,
    nickname: nickname,
    role: role ?? this.role,
    isReady: isReady ?? this.isReady,
    connected: connected,
  );

  static RoomMember fromJson(Object? json) {
    final m = _map(json);
    return RoomMember(
      userId: _string(m, 'user_id'),
      nickname: _string(m, 'nickname'),
      role: switch (m['role']) {
        'player' => MemberRole.player,
        'spectator' => MemberRole.spectator,
        _ => throw _invalid('role'),
      },
      isReady: _bool(m, 'is_ready'),
      connected: _bool(m, 'connected'),
    );
  }

  @override
  bool operator ==(Object other) =>
      other is RoomMember &&
      other.userId == userId &&
      other.nickname == nickname &&
      other.role == role &&
      other.isReady == isReady &&
      other.connected == connected;

  @override
  int get hashCode => Object.hash(userId, nickname, role, isReady, connected);
}

/// Oda görünümü (`RoomView`): REST yanıtları ve WS anlık görüntüsü.
@immutable
class RoomView {
  const RoomView({
    required this.id,
    required this.inviteCode,
    required this.status,
    required this.hostUserId,
    required this.capacity,
    required this.currentGameId,
    required this.version,
    required this.members,
    this.scenarioId,
  });

  final String id;
  final String inviteCode;
  final RoomStatus status;
  final String hostUserId;
  final int capacity;

  /// Oda sahibinin lobide seçtiği senaryo; `null` = rastgele.
  final String? scenarioId;
  final String? currentGameId;

  /// Lobi olaylarının sürümü (WS `state_version` ile aynı sayaç).
  final int version;

  /// Aktif üyeler, katılma sırasıyla.
  final List<RoomMember> members;

  /// Oyuncular (izleyiciler hariç).
  Iterable<RoomMember> get players =>
      members.where((m) => m.role == MemberRole.player);

  RoomMember? memberOf(String? userId) {
    for (final m in members) {
      if (m.userId == userId) return m;
    }
    return null;
  }

  RoomView copyWith({
    RoomStatus? status,
    String? hostUserId,
    int? capacity,
    String? Function()? scenarioId,
    String? Function()? currentGameId,
    int? version,
    List<RoomMember>? members,
  }) => RoomView(
    id: id,
    inviteCode: inviteCode,
    status: status ?? this.status,
    hostUserId: hostUserId ?? this.hostUserId,
    capacity: capacity ?? this.capacity,
    scenarioId: scenarioId != null ? scenarioId() : this.scenarioId,
    currentGameId: currentGameId != null ? currentGameId() : this.currentGameId,
    version: version ?? this.version,
    members: members != null ? List.unmodifiable(members) : this.members,
  );

  /// Bozuk yanıtta `ApiException(invalidResponse)` fırlatır.
  static RoomView fromJson(Object? json) {
    final m = _map(json);
    final members = m['members'];
    if (members is! List<Object?>) throw _invalid('members');
    final game = m['current_game_id'];
    if (game != null && game is! String) throw _invalid('current_game_id');
    final scenario = m['scenario_id'];
    if (scenario != null && scenario is! String) throw _invalid('scenario_id');
    return RoomView(
      id: _string(m, 'id'),
      inviteCode: _string(m, 'invite_code'),
      status: switch (m['status']) {
        'lobby' => RoomStatus.lobby,
        'in_game' => RoomStatus.inGame,
        'finished' => RoomStatus.finished,
        _ => throw _invalid('status'),
      },
      hostUserId: _string(m, 'host_user_id'),
      capacity: _int(m, 'capacity'),
      scenarioId: scenario as String?,
      currentGameId: game as String?,
      version: _int(m, 'version'),
      members: List.unmodifiable(members.map(RoomMember.fromJson)),
    );
  }
}

/// Lobideki senaryo seçeneği (`GET /v1/scenarios`).
@immutable
class ScenarioOption {
  const ScenarioOption({required this.id, required this.title});

  final String id;
  final String title;

  static List<ScenarioOption> listFromJson(Object? json) {
    final m = _map(json);
    final list = m['scenarios'];
    if (list is! List<Object?>) throw _invalid('scenarios');
    return List.unmodifiable(
      list.map((e) {
        final o = _map(e);
        return ScenarioOption(id: _string(o, 'id'), title: _string(o, 'title'));
      }),
    );
  }

  @override
  bool operator ==(Object other) =>
      other is ScenarioOption && other.id == id && other.title == title;

  @override
  int get hashCode => Object.hash(id, title);
}

Map<String, Object?> _map(Object? v) {
  if (v is Map<String, Object?>) return v;
  throw _invalid('nesne bekleniyordu');
}

String _string(Map<String, Object?> m, String key) {
  final v = m[key];
  if (v is String && v.isNotEmpty) return v;
  throw _invalid(key);
}

int _int(Map<String, Object?> m, String key) {
  final v = m[key];
  if (v is int) return v;
  throw _invalid(key);
}

bool _bool(Map<String, Object?> m, String key) {
  final v = m[key];
  if (v is bool) return v;
  throw _invalid(key);
}

ApiException _invalid(String what) => ApiException(
  kind: ApiErrorKind.invalidResponse,
  debugMessage: 'oda yanıtı: $what',
);
