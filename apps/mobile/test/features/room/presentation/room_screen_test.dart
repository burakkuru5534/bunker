import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/realtime/realtime_client_provider.dart';
import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/session_store.dart';
import 'package:siginak/features/room/room.dart';
import 'package:siginak/features/session/session.dart';

import '../../../helpers/fake_backend.dart';
import '../../../helpers/fake_http_adapter.dart';
import '../../../helpers/fake_ws.dart';
import '../../../helpers/pump_app.dart';

void main() {
  const roomId = '44444444-4444-4444-8444-444444444444';
  late FakeBackend backend;
  late FakeWsConnector ws;
  late int exits;
  late List<String> requests;

  setUp(() {
    backend = FakeBackend()..activeRoomId = roomId;
    ws = FakeWsConnector();
    exits = 0;
    requests = [];
    for (final (m, p) in [
      ('POST', 'ready'),
      ('POST', 'kick'),
      ('POST', 'start'),
      ('POST', 'leave'),
    ]) {
      backend.on(m, '/v1/rooms/$roomId/$p', (r) async {
        requests.add('$p ${jsonEncode(r.data)}');
        return ResponseBodyNoContent.value;
      });
    }
    backend.on('PATCH', '/v1/rooms/$roomId/settings', (r) async {
      requests.add('settings ${jsonEncode(r.data)}');
      return jsonOk(
        snapshotPayload(
          roomId: roomId,
          me: 'u-guest',
          members: [wsMember('u-guest', 'Ayşe')],
        )['room'],
      );
    });
  });

  Future<void> frames(WidgetTester tester, [int n = 10]) async {
    for (var i = 0; i < n; i++) {
      await tester.pump(const Duration(milliseconds: 50));
    }
  }

  Future<ProviderContainer> pump(WidgetTester tester) async {
    // Uzun bir telefon ekranı (412×915): tüm lobi görünür.
    tester.view
      ..physicalSize = const Size(1236, 2745)
      ..devicePixelRatio = 3;
    addTearDown(tester.view.reset);
    final store = InMemorySecureStore();
    await SessionStore(store).save(
      StoredSession(
        token: 't',
        expiresAt: DateTime.now().add(const Duration(days: 30)),
        userId: 'u-guest',
        nickname: 'Ayşe',
      ),
    );
    await pumpWithProviders(
      tester,
      RoomScreen(roomId: roomId, onExit: () => exits++),
      secureStore: store,
      overrides: [
        apiHttpAdapterProvider.overrideWithValue(backend.adapter),
        wsConnectorProvider.overrideWithValue(ws.call),
      ],
    );
    await frames(tester);
    return ProviderScope.containerOf(tester.element(find.byType(RoomScreen)));
  }

  /// Snapshot gönderir; varsayılan: ben (u-guest) oda sahibiyim, 2 oyuncu.
  Future<void> snapshot(
    WidgetTester tester, {
    List<Map<String, Object?>>? members,
    String host = 'u-guest',
    int capacity = 2,
  }) async {
    ws.last.serverSend(
      'session.snapshot',
      payload: snapshotPayload(
        roomId: roomId,
        me: 'u-guest',
        host: host,
        capacity: capacity,
        version: 5,
        members:
            members ??
            [
              wsMember('u-guest', 'Ayşe'),
              wsMember('u2', 'Mehmet', ready: true),
            ],
      ),
    );
    await frames(tester);
  }

  List<Map<String, Object?>> fourReady() => [
    wsMember('u-guest', 'Ayşe', ready: true),
    for (var i = 2; i <= 4; i++) wsMember('u$i', 'Oyuncu $i', ready: true),
  ];

  testWidgets('Snapshot gelene kadar bağlanılıyor; sonra lobi', (tester) async {
    await pump(tester);
    expect(find.text('Odaya bağlanılıyor…'), findsOneWidget);

    await snapshot(tester);

    expect(find.text('K7M2QX'), findsOneWidget);
    expect(find.text('Oyuncular (2/10)'), findsOneWidget);
    expect(find.text('Ayşe (sen)'), findsOneWidget);
    expect(find.text('Mehmet'), findsOneWidget);
    expect(find.text('Oda sahibi'), findsOneWidget);
    expect(find.text('Hazır'), findsOneWidget);
    expect(find.text('Bekliyor'), findsOneWidget);
  });

  testWidgets('Kod kopyalanır', (tester) async {
    final copied = <String>[];
    tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
      SystemChannels.platform,
      (call) async {
        if (call.method == 'Clipboard.setData') {
          copied.add((call.arguments as Map)['text'] as String);
        }
        return null;
      },
    );
    await pump(tester);
    await snapshot(tester);

    await tester.tap(find.text('Kopyala'));
    await frames(tester);

    expect(copied, ['K7M2QX']);
    expect(find.text('Davet kodu kopyalandı'), findsOneWidget);
  });

  testWidgets('Sahip: başlatma nedeni yazılı; az oyuncuyla kapalı', (
    tester,
  ) async {
    await pump(tester);
    await snapshot(tester);

    expect(
      find.text('Başlatmak için en az 4 oyuncu gerekli (2/4)'),
      findsOneWidget,
    );
    final start = tester.widget<FilledButton>(find.byType(FilledButton));
    expect(start.onPressed, isNull);
  });

  testWidgets('Sahip: herkes hazırsa başlatır; hata snackbar', (tester) async {
    await pump(tester);
    await snapshot(tester, members: fourReady());

    await tester.tap(find.text('Başlat'));
    await frames(tester);
    expect(requests, ['start null']);

    backend.on(
      'POST',
      '/v1/rooms/$roomId/start',
      (r) async => jsonResponse(
        apiError(
          'START_CONDITIONS_NOT_MET',
          details: {'reason': 'not_all_ready'},
        ),
        409,
      ),
    );
    await tester.tap(find.text('Başlat'));
    await frames(tester);
    expect(
      find.text('Başlatmak için 4–10 oyuncu ve herkesin hazır olması gerekir.'),
      findsOneWidget,
    );
  });

  testWidgets('Hazırım anahtarı istek gönderir; olayla güncellenir', (
    tester,
  ) async {
    await pump(tester);
    await snapshot(tester);

    await tester.tap(find.byType(Switch));
    await frames(tester);
    expect(requests, ['ready {"ready":true}']);

    ws.last.serverSend(
      'room.ready_changed',
      stateVersion: 6,
      payload: {'user_id': 'u-guest', 'is_ready': true},
    );
    await frames(tester);
    expect(tester.widget<Switch>(find.byType(Switch)).value, isTrue);
    expect(find.text('Hazır'), findsNWidgets(2));
  });

  testWidgets('Sahip kapasiteyi değiştirir', (tester) async {
    await pump(tester);
    await snapshot(tester);

    await tester.tap(find.text('1 kişi'));
    await frames(tester);

    expect(requests, ['settings {"capacity":1}']);
  });

  testWidgets('Sahip oyuncu atar (onaylı)', (tester) async {
    await pump(tester);
    await snapshot(tester);

    await tester.tap(find.byTooltip('Odadan at'));
    await frames(tester);
    expect(find.text('Mehmet odadan atılsın mı?'), findsOneWidget);

    await tester.tap(find.text('At'));
    await frames(tester);
    expect(requests, ['kick {"user_id":"u2"}']);
  });

  testWidgets('Sahip olmayan: atma ve başlatma yok, kapasite kilitli', (
    tester,
  ) async {
    await pump(tester);
    await snapshot(tester, host: 'u2');

    expect(find.byTooltip('Odadan at'), findsNothing);
    expect(find.text('Başlat'), findsNothing);
    expect(
      find.text('Oda sahibinin oyunu başlatması bekleniyor.'),
      findsOneWidget,
    );
    await tester.tap(find.text('1 kişi'));
    await frames(tester);
    expect(requests, isEmpty);
  });

  testWidgets('Ayrıl: onay → istek → ana ekrana, aktif oda temizlenir', (
    tester,
  ) async {
    final c = await pump(tester);
    await snapshot(tester);
    c.read(sessionControllerProvider.notifier).setActiveRoom(roomId);

    await tester.tap(find.byTooltip('Odadan ayrıl'));
    await frames(tester);
    expect(find.text('Odadan ayrılmak istiyor musun?'), findsOneWidget);
    await tester.tap(find.text('Vazgeç'));
    await frames(tester);
    expect(requests, isEmpty);

    await tester.tap(find.byTooltip('Odadan ayrıl'));
    await frames(tester);
    await tester.tap(find.text('Ayrıl'));
    await frames(tester);

    expect(requests, ['leave null']);
    expect(exits, 1);
    expect(c.read(sessionControllerProvider).activeRoomId, isNull);
  });

  testWidgets('Geri tuşu ayrılma onayı sorar', (tester) async {
    await pump(tester);
    await snapshot(tester);

    final popped = await tester.binding.handlePopRoute();
    await frames(tester);

    expect(popped, isTrue); // uygulama kapanmadı
    expect(find.text('Odadan ayrılmak istiyor musun?'), findsOneWidget);
  });

  testWidgets('Atılınca ana ekrana döner ve bildirilir', (tester) async {
    await pump(tester);
    await snapshot(tester);

    ws.last.serverSend(
      'room.member_kicked',
      stateVersion: 6,
      payload: {'user_id': 'u-guest'},
    );
    await frames(tester);

    expect(exits, 1);
    expect(
      find.text('Bu odadan çıkarıldın, tekrar katılamazsın.'),
      findsOneWidget,
    );
  });

  testWidgets('Bağlantı koparsa şerit; dönünce kaybolur', (tester) async {
    await pump(tester);
    await snapshot(tester);

    ws.last.serverClose(1006);
    await frames(tester, 2);
    expect(find.text('Bağlantı koptu, yeniden bağlanıyoruz…'), findsOneWidget);

    await frames(tester, 20); // yeniden bağlanma (≈0,5 sn)
    expect(ws.sockets, hasLength(2));
    await snapshot(tester);
    expect(find.text('Bağlantı koptu, yeniden bağlanıyoruz…'), findsNothing);
  });

  testWidgets('Başka cihazda açılınca durur; yeniden bağlanılabilir', (
    tester,
  ) async {
    await pump(tester);
    await snapshot(tester);

    ws.last.serverClose(4008);
    await frames(tester);
    expect(
      find.text('Bu oda başka bir cihazda ya da pencerede açıldı.'),
      findsOneWidget,
    );
    expect(ws.sockets, hasLength(1));

    await tester.tap(find.text('Tekrar dene'));
    await frames(tester);
    expect(ws.sockets, hasLength(2));
  });

  testWidgets('Oyun başlayınca lobi kapanır', (tester) async {
    await pump(tester);
    await snapshot(tester, members: fourReady());

    ws.last.serverSend(
      'game.started',
      stateVersion: 1,
      payload: {'game_id': 'g1'},
    );
    await frames(tester);

    expect(find.text('Oyun başladı.'), findsOneWidget);
    expect(find.text('K7M2QX'), findsNothing);
  });
}
