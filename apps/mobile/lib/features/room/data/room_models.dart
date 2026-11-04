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
  });

  final String id;
  final String inviteCode;
  final RoomStatus status;
  final String hostUserId;
  final int capacity;
  final String? currentGameId;

  /// Lobi olaylarının sürümü (WS `state_version` ile aynı sayaç).
  final int version;

  /// Aktif üyeler, katılma sırasıyla.
  final List<RoomMember> members;

  /// Bozuk yanıtta `ApiException(invalidResponse)` fırlatır.
  static RoomView fromJson(Object? json) {
    final m = _map(json);
    final members = m['members'];
    if (members is! List<Object?>) throw _invalid('members');
    final game = m['current_game_id'];
    if (game != null && game is! String) throw _invalid('current_game_id');
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
      currentGameId: game as String?,
      version: _int(m, 'version'),
      members: List.unmodifiable(members.map(RoomMember.fromJson)),
    );
  }
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
