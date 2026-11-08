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

  /// Hazır durumu (yalnızca oyuncu, lobide). Sonuç `room.ready_changed` ile gelir.
  Future<void> setReady(String roomId, {required bool ready}) async {
    await _api.post('/v1/rooms/$roomId/ready', body: {'ready': ready});
  }

  /// Kapasite (oda sahibi, lobide).
  Future<RoomView> setCapacity(String roomId, int capacity) async =>
      RoomView.fromJson(
        await _api.patch(
          '/v1/rooms/$roomId/settings',
          body: {'capacity': capacity},
        ),
      );

  /// Senaryo seçimi (oda sahibi, lobide); `null` = rastgele.
  Future<RoomView> setScenario(String roomId, String? scenarioId) async =>
      RoomView.fromJson(
        await _api.patch(
          '/v1/rooms/$roomId/settings',
          body: {'scenario_id': scenarioId},
        ),
      );

  /// Seçilebilir senaryolar (aktif katalog).
  Future<List<ScenarioOption>> scenarios() async =>
      ScenarioOption.listFromJson(await _api.get('/v1/scenarios'));

  /// Üyeyi odadan atar (oda sahibi, lobide).
  Future<void> kick(String roomId, String userId) async {
    await _api.post('/v1/rooms/$roomId/kick', body: {'user_id': userId});
  }

  /// Oyunu başlatır (oda sahibi). Ekran `game.*` olaylarıyla güncellenir.
  Future<void> start(String roomId) async {
    await _api.post('/v1/rooms/$roomId/start');
  }

  /// Tekrar oyna (oda sahibi, oyun bitince): oda lobiye döner.
  Future<RoomView> rematch(String roomId) async =>
      RoomView.fromJson(await _api.post('/v1/rooms/$roomId/rematch'));

  /// İdempotent: üye değilse de başarılıdır.
  Future<void> leave(String roomId) async {
    await _api.post('/v1/rooms/$roomId/leave');
  }
}

final roomRepositoryProvider = Provider<RoomRepository>(
  (ref) => RoomRepository(ref.watch(apiClientProvider)),
);

/// Lobideki senaryo listesi (oturum boyunca bir kez yüklenir; hata olursa
/// yeniden denenir).
final scenarioOptionsProvider = FutureProvider<List<ScenarioOption>>(
  (ref) => ref.watch(roomRepositoryProvider).scenarios(),
);
