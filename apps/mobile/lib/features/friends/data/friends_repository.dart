import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/network/api_exception.dart';

/// Listelerdeki oyuncu (arkadaş, gelen/giden istek).
@immutable
class FriendPerson {
  const FriendPerson({required this.userId, required this.nickname});

  final String userId;
  final String nickname;

  static FriendPerson fromJson(Object? json) {
    final m = _map(json);
    return FriendPerson(
      userId: _string(m, 'user_id'),
      nickname: _string(m, 'nickname'),
    );
  }

  @override
  bool operator ==(Object other) =>
      other is FriendPerson &&
      other.userId == userId &&
      other.nickname == nickname;

  @override
  int get hashCode => Object.hash(userId, nickname);
}

/// `GET /v1/friends`.
@immutable
class FriendsData {
  const FriendsData({
    required this.friends,
    required this.incoming,
    required this.outgoing,
  });

  final List<FriendPerson> friends;
  final List<FriendPerson> incoming;
  final List<FriendPerson> outgoing;

  static FriendsData fromJson(Object? json) {
    final m = _map(json);
    List<FriendPerson> list(String k) {
      final v = m[k];
      if (v is! List<Object?>) throw _invalid(k);
      return List.unmodifiable(v.map(FriendPerson.fromJson));
    }

    return FriendsData(
      friends: list('friends'),
      incoming: list('incoming'),
      outgoing: list('outgoing'),
    );
  }
}

/// Lobi daveti (`GET /v1/inbox`).
@immutable
class RoomInvite {
  const RoomInvite({
    required this.roomId,
    required this.inviteCode,
    required this.from,
    required this.players,
  });

  final String roomId;
  final String inviteCode;
  final FriendPerson from;
  final int players;

  static RoomInvite fromJson(Object? json) {
    final m = _map(json);
    final players = m['players'];
    if (players is! int) throw _invalid('players');
    return RoomInvite(
      roomId: _string(m, 'room_id'),
      inviteCode: _string(m, 'invite_code'),
      from: FriendPerson.fromJson(m['from']),
      players: players,
    );
  }
}

/// Ana ekranın yokladığı özet: davetler ve gelen istek sayısı.
@immutable
class Inbox {
  const Inbox({required this.invites, required this.incomingRequests});

  static const empty = Inbox(invites: [], incomingRequests: 0);

  final List<RoomInvite> invites;
  final int incomingRequests;

  static Inbox fromJson(Object? json) {
    final m = _map(json);
    final invites = m['invites'];
    final n = m['incoming_requests'];
    if (invites is! List<Object?>) throw _invalid('invites');
    if (n is! int) throw _invalid('incoming_requests');
    return Inbox(
      invites: List.unmodifiable(invites.map(RoomInvite.fromJson)),
      incomingRequests: n,
    );
  }
}

/// İstek sonucu: istek gönderildi ya da karşılıklı istekle arkadaş olundu.
enum FriendRequestOutcome { requested, friends }

/// Arkadaşlık ve davet uç noktaları (docs/api.md "Arkadaşlar ve davetler").
class FriendsRepository {
  FriendsRepository(this._api);

  final ApiClient _api;

  Future<FriendsData> list() async =>
      FriendsData.fromJson(await _api.get('/v1/friends'));

  Future<(FriendRequestOutcome, FriendPerson)> request(String nickname) async {
    final m = _map(
      await _api.post('/v1/friends/requests', body: {'nickname': nickname}),
    );
    final outcome = switch (m['status']) {
      'friends' => FriendRequestOutcome.friends,
      'requested' => FriendRequestOutcome.requested,
      _ => throw _invalid('status'),
    };
    return (outcome, FriendPerson.fromJson(m['user']));
  }

  Future<void> accept(String userId) async {
    await _api.post('/v1/friends/requests/$userId/accept');
  }

  /// Gelen isteği reddeder ya da gönderileni geri çeker.
  Future<void> deleteRequest(String userId) async {
    await _api.delete('/v1/friends/requests/$userId');
  }

  Future<void> remove(String userId) async {
    await _api.delete('/v1/friends/$userId');
  }

  Future<void> invite(String roomId, String userId) async {
    await _api.post('/v1/rooms/$roomId/invites', body: {'user_id': userId});
  }

  Future<Inbox> inbox() async => Inbox.fromJson(await _api.get('/v1/inbox'));

  Future<void> decline(String roomId) async {
    await _api.delete('/v1/invites/$roomId');
  }
}

final friendsRepositoryProvider = Provider<FriendsRepository>(
  (ref) => FriendsRepository(ref.watch(apiClientProvider)),
);

/// Arkadaş listesi (ekranlar `ref.invalidate` ile tazeler).
final friendsProvider = FutureProvider.autoDispose<FriendsData>(
  (ref) => ref.watch(friendsRepositoryProvider).list(),
);

/// Ana ekran özeti; ana ekran açıkken düzenli aralıkla tazelenir.
final inboxProvider = FutureProvider.autoDispose<Inbox>(
  (ref) => ref.watch(friendsRepositoryProvider).inbox(),
);

Map<String, Object?> _map(Object? v) {
  if (v is Map<String, Object?>) return v;
  throw _invalid('nesne bekleniyordu');
}

String _string(Map<String, Object?> m, String key) {
  final v = m[key];
  if (v is String) return v;
  throw _invalid(key);
}

ApiException _invalid(String what) => ApiException(
  kind: ApiErrorKind.invalidResponse,
  debugMessage: 'arkadaş yanıtı: $what',
);
