import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/features/room/data/room_models.dart';

/// Oda uç noktaları (docs/api.md "Oda").
class RoomRepository {
  RoomRepository(this._api);

  final ApiClient _api;

  /// Oda kurar; kuran oda sahibi ve oyuncudur. Kapasite sunucuda varsayılan.
  Future<RoomView> create() async =>
      RoomView.fromJson(await _api.post('/v1/rooms', body: {}));

  /// Davet koduyla katılır (zaten üyeyse aynı oda döner).
  Future<RoomView> join(String inviteCode) async => RoomView.fromJson(
    await _api.post('/v1/rooms/join', body: {'invite_code': inviteCode}),
  );

  Future<RoomView> fetch(String roomId) async =>
      RoomView.fromJson(await _api.get('/v1/rooms/$roomId'));

  /// İdempotent: üye değilse de başarılıdır.
  Future<void> leave(String roomId) async {
    await _api.post('/v1/rooms/$roomId/leave');
  }
}

final roomRepositoryProvider = Provider<RoomRepository>(
  (ref) => RoomRepository(ref.watch(apiClientProvider)),
);
