import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/realtime/realtime_client_provider.dart';
import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/session_store.dart';
import 'package:siginak/core/ui/app_buttons.dart';
import 'package:siginak/features/room/room.dart';
import 'package:siginak/features/session/session.dart';

import '../../../helpers/chat.dart';
import '../../../helpers/fake_backend.dart';
import '../../../helpers/fake_http_adapter.dart';
import '../../../helpers/fake_intro_audio.dart';
import '../../../helpers/fake_ws.dart';
import '../../../helpers/pump_app.dart';
import '../../../helpers/small_screen.dart';

void main() {
  const roomId = '44444444-4444-4444-8444-444444444444';
  late FakeBackend backend;
  late FakeWsConnector ws;
  late int exits;
  late List<String> requests;
  late List<FakeLoopingMusic> music;
  late bool scenariosFail;
  late int scenarioCalls;

  setUp(() {
    music = [];
    scenariosFail = false;
    scenarioCalls = 0;
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
    backend.on('GET', '/v1/scenarios', (r) async {
      scenarioCalls++;
      if (scenariosFail) return jsonResponse(apiError('INTERNAL'), 500);
      return jsonOk({
        'scenarios': [
          {'id': 'meteor', 'title': 'Göktaşı'},
          {'id': 'nuclear_winter', 'title': 'Nükleer Kış'},
        ],
      });
    });
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
      lobbyMusic: () => FakeLoopingMusic(music),
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
    String? scenarioId,
  }) async {
    ws.last.serverSend(
      'session.snapshot',
      payload: snapshotPayload(
        roomId: roomId,
        me: 'u-guest',
        host: host,
        capacity: capacity,
        scenarioId: scenarioId,
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

  testWidgets('320 dp ve %200 yazı: lobi ve ayrılma onayı taşmaz', (
    tester,
  ) async {
    await pump(tester);
    useSmallScreen(tester);
    await snapshot(tester, members: fourReady());
    expect(tester.takeException(), isNull);
    await tester.tap(find.byTooltip('Odadan ayrıl'));
    await frames(tester);
    expect(tester.takeException(), isNull);
  });

  testWidgets('Lobi dokunma hedefleri ve etiketler (360 dp)', (tester) async {
    await pump(tester);
    useSmallScreen(tester, size: const Size(360, 640), textScale: 1);
    await snapshot(tester, members: fourReady());
    await expectAccessibleTapTargets(tester);
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

  group('Senaryo seçimi', () {
    final picker = find.byKey(const ValueKey('scenario-picker'));

    testWidgets('sahip listeden seçer; olayla güncellenir', (tester) async {
      await pump(tester);
      await snapshot(tester);
      expect(find.text('Felaket senaryosu'), findsOneWidget);
      expect(
        find.descendant(of: picker, matching: find.text('Rastgele')),
        findsOneWidget,
      );

      await tester.tap(picker);
      await frames(tester);
      expect(find.text('Senaryo seç'), findsOneWidget);
      expect(find.text('Nükleer Kış'), findsOneWidget);
      await tester.tap(find.text('Göktaşı'));
      await frames(tester);
      expect(requests.last, 'settings {"scenario_id":"meteor"}');

      ws.last.serverSend(
        'room.settings_changed',
        stateVersion: 6,
        payload: {'capacity': 2, 'scenario_id': 'meteor'},
      );
      await frames(tester);
      expect(
        find.descendant(of: picker, matching: find.text('Göktaşı')),
        findsOneWidget,
      );

      // Rastgeleye dönüş.
      await tester.tap(picker);
      await frames(tester);
      await tester.tap(find.byKey(const ValueKey('scenario-option-random')));
      await frames(tester);
      expect(requests.last, 'settings {"scenario_id":null}');

      // Aynı seçeneğe dokunmak istek göndermez.
      final n = requests.length;
      await tester.tap(picker);
      await frames(tester);
      await tester.tap(find.byKey(const ValueKey('scenario-option-meteor')));
      await frames(tester);
      expect(requests, hasLength(n));
    });

    testWidgets('oda sahibi değilse yalnızca seçimi görür', (tester) async {
      await pump(tester);
      await snapshot(tester, host: 'u2', scenarioId: 'nuclear_winter');
      expect(
        find.descendant(of: picker, matching: find.text('Nükleer Kış')),
        findsOneWidget,
      );
      expect(tester.widget<OutlinedButton>(picker).onPressed, isNull);
    });

    testWidgets('liste yüklenemezse sahip tekrar deneyebilir', (tester) async {
      scenariosFail = true;
      await pump(tester);
      await snapshot(tester);
      expect(find.text('Senaryolar yüklenemedi.'), findsOneWidget);

      scenariosFail = false;
      final before = scenarioCalls;
      await tester.tap(find.text('Tekrar dene'));
      await frames(tester);
      expect(find.text('Senaryolar yüklenemedi.'), findsNothing);
      expect(picker, findsOneWidget);
      expect(scenarioCalls, greaterThan(before));
    });
  });

  group('Lobi müziği', () {
    testWidgets('lobide çalar; ses düğmesi sessize alır', (tester) async {
      await pump(tester);
      await snapshot(tester);
      expect(music.single.calls, ['muted:false', 'play']);

      await tester.tap(find.byKey(const ValueKey('sound-toggle')));
      await frames(tester);
      expect(music.single.calls.last, 'muted:true');
      expect(find.byTooltip('Sesi aç'), findsOneWidget);

      // Uygulama arka planda duraklar, öne gelince sürer.
      final binding = tester.binding;
      for (final s in [AppLifecycleState.inactive, AppLifecycleState.hidden]) {
        binding.handleAppLifecycleStateChanged(s);
      }
      expect(music.single.calls.last, 'pause');
      for (final s in [AppLifecycleState.inactive, AppLifecycleState.resumed]) {
        binding.handleAppLifecycleStateChanged(s);
      }
      expect(music.single.calls.last, 'play');
    });

    testWidgets('oyun başlayınca durur', (tester) async {
      await pump(tester);
      await snapshot(tester, members: fourReady());
      expect(music, hasLength(1));
      ws.last.serverSend(
        'game.started',
        stateVersion: 1,
        payload: gameStartedPayload([
          ('u-guest', 'Ayşe'),
          ('u2', 'Oyuncu 2'),
          ('u3', 'Oyuncu 3'),
          ('u4', 'Oyuncu 4'),
        ]),
      );
      await frames(tester);
      expect(music.single.disposed, isTrue);
      expect(find.byKey(const ValueKey('sound-toggle')), findsNothing);
    });
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

  group('yeniden bağlanma (PLAY-009)', () {
    const reconnecting = 'Bağlantı koptu, yeniden bağlanıyoruz…';

    testWidgets(
      'Uzun kopmada "Bağlantı yok" + Tekrar dene; dönünce "Bağlandı"',
      (tester) async {
        await pump(tester);
        await snapshot(tester);
        expect(
          find.text('Bağlandı'),
          findsNothing,
        ); // ilk bağlantıda gösterilmez

        ws.failNext = 1000;
        ws.last.serverClose(1006);
        await frames(tester, 2);
        expect(find.text(reconnecting), findsOneWidget);

        await tester.pump(const Duration(seconds: 9));
        expect(find.text('Bağlantı yok'), findsNothing);
        await tester.pump(const Duration(seconds: 2));
        expect(find.text('Bağlantı yok'), findsOneWidget);
        expect(find.text(reconnecting), findsNothing);

        // Tekrar dene beklemeyi atlar: hemen yeni deneme.
        ws.failNext = 0;
        final attempts = ws.uris.length;
        await tester.tap(find.text('Tekrar dene'));
        await frames(tester, 2);
        expect(ws.uris.length, attempts + 1);
        expect(ws.sockets, hasLength(2));
        await snapshot(tester);
        expect(find.text('Bağlandı'), findsOneWidget);
        expect(find.text('Bağlantı yok'), findsNothing);

        await tester.pump(const Duration(seconds: 3));
        expect(find.text('Bağlandı'), findsNothing);
      },
    );

    testWidgets('Tekrar dene başarısızsa yeniden turuncu, sonra kırmızı', (
      tester,
    ) async {
      await pump(tester);
      await snapshot(tester);
      ws.failNext = 1000;
      ws.last.serverClose(1006);
      await frames(tester, 2);
      await tester.pump(const Duration(seconds: 11));
      await tester.tap(find.text('Tekrar dene'));
      await frames(tester, 2);
      expect(find.text(reconnecting), findsOneWidget);
      await tester.pump(const Duration(seconds: 11));
      expect(find.text('Bağlantı yok'), findsOneWidget);
    });

    testWidgets('Lobide bağlantı yokken eylemler kapalı', (tester) async {
      await pump(tester);
      await snapshot(tester, members: fourReady());
      VoidCallback? start() => tester
          .widget<AppButton>(find.widgetWithText(AppButton, 'Başlat'))
          .onPressed;
      expect(start(), isNotNull);
      expect(find.byTooltip('Odadan at'), findsWidgets);

      ws.last.serverClose(1006);
      await frames(tester, 2);
      expect(start(), isNull);
      expect(
        tester.widget<SwitchListTile>(find.byType(SwitchListTile)).onChanged,
        isNull,
      );
      expect(find.byTooltip('Odadan at'), findsNothing);

      await frames(tester, 20);
      await snapshot(tester, members: fourReady());
      expect(start(), isNotNull);
    });
  });

  testWidgets('Oda yüklenemezse hata + Tekrar dene (PLAY-011)', (tester) async {
    await pump(tester);
    for (var i = 0; i < 3; i++) {
      ws.last.serverSend(
        'session.snapshot',
        payload: snapshotPayload(
          roomId: roomId,
          me: 'u-guest',
          status: 'in_game',
          members: [wsMember('u-guest', 'Ayşe')],
        ),
      );
      await frames(tester, 2);
      if (i < 2) expect(find.text('Odaya bağlanılıyor…'), findsOneWidget);
    }
    expect(
      find.text(
        'Oda yüklenemedi. Tekrar dene; sorun sürerse odadan çıkıp yeniden katıl.',
      ),
      findsOneWidget,
    );
    expect(find.byTooltip('Odadan ayrıl'), findsOneWidget); // çıkış yolu açık

    await tester.tap(find.text('Tekrar dene'));
    await frames(tester, 2);
    expect(find.text('Odaya bağlanılıyor…'), findsOneWidget);
    expect(ws.last.sent.last, contains('"sync"'));
    await snapshot(tester);
    expect(find.text('K7M2QX'), findsOneWidget);
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

  testWidgets('Oyun başlayınca lobi kapanır, senaryo tanıtımı gelir', (
    tester,
  ) async {
    await pump(tester);
    await snapshot(tester, members: fourReady());

    ws.last.serverSend(
      'game.started',
      stateVersion: 1,
      payload: gameStartedPayload([
        ('u-guest', 'Ayşe'),
        ('u2', 'Oyuncu 2'),
        ('u3', 'Oyuncu 3'),
        ('u4', 'Oyuncu 4'),
      ]),
    );
    await frames(tester);
    expect(find.text('K7M2QX'), findsNothing);
    expect(find.text('Kartlar dağıtılıyor…'), findsOneWidget);
    expect(find.text('Oyun'), findsOneWidget); // başlık

    ws.last
      ..serverSend(
        'game.your_cards',
        stateVersion: 1,
        payload: {'cards': myCardsJson()},
      )
      ..serverSend(
        'game.phase_changed',
        stateVersion: 1,
        payload: {
          'phase': 'SCENARIO_INTRO',
          'round': 0,
          'phase_deadline_at': DateTime.now()
              .add(const Duration(seconds: 15))
              .toUtc()
              .toIso8601String(),
          'server_time': DateTime.now().toUtc().toIso8601String(),
          'turn_user_id': null,
        },
      );
    await frames(tester);

    expect(find.text('Göktaşı'), findsWidgets);
    expect(find.text('Sığınağa yalnızca 2 kişi girebilir.'), findsOneWidget);
    expect(find.byKey(const ValueKey('phase-countdown')), findsOneWidget);
  });

  testWidgets('Oyun sürerken ayrılma uyarısı farklıdır', (tester) async {
    await pump(tester);
    await snapshot(tester, members: fourReady());
    ws.last.serverSend(
      'game.started',
      stateVersion: 1,
      payload: gameStartedPayload([('u-guest', 'Ayşe'), ('u2', 'Mehmet')]),
    );
    await frames(tester);

    await tester.tap(find.byTooltip('Odadan ayrıl'));
    await frames(tester);

    expect(
      find.text(
        'Oyun sürerken ayrılırsan elenmiş sayılırsın ve tüm kartların açılır.',
      ),
      findsOneWidget,
    );
  });

  group('oyun eylemleri', () {
    final gameMembers = [
      wsMember('u-guest', 'Ayşe'),
      wsMember('u2', 'Mehmet'),
      wsMember('u3', 'Zeynep'),
    ];
    final gameParticipants = [
      participantJson('u-guest', 'Ayşe', 0),
      participantJson('u2', 'Mehmet', 1),
      participantJson('u3', 'Zeynep', 2),
    ];

    Future<void> gameSnapshot(WidgetTester tester, String phase) async {
      ws.last.serverSend(
        'session.snapshot',
        payload: snapshotPayload(
          roomId: roomId,
          me: 'u-guest',
          status: 'in_game',
          members: gameMembers,
          game: snapshotGame(
            participants: gameParticipants,
            phase: phase,
            turnUserId: phase == 'REVEAL' ? 'u-guest' : null,
          ),
        ),
      );
      await frames(tester);
    }

    testWidgets('Oyunda sohbet Kartlarım yanında; çubukta değil', (
      tester,
    ) async {
      await pump(tester);
      await gameSnapshot(tester, 'DISCUSSION');
      expect(find.byTooltip('Sohbet'), findsNothing);
      await tester.tap(find.byKey(const ValueKey('discussion-chat')));
      await frames(tester);
      expect(find.byType(TextField), findsOneWidget); // sohbet paneli açık
    });

    testWidgets('Bağlantı yokken eylemler kapalı; oyunda uyarı daha açık', (
      tester,
    ) async {
      await pump(tester);
      await gameSnapshot(tester, 'DISCUSSION');
      await tester.pump(const Duration(seconds: 21)); // Hazırım kilidi
      VoidCallback? ready() => tester
          .widget<AppButton>(find.widgetWithText(AppButton, 'Hazırım'))
          .onPressed;
      expect(ready(), isNotNull);

      ws.failNext = 1000;
      ws.last.serverClose(1006);
      await frames(tester, 2);
      expect(find.text('Hazırım'), findsOneWidget); // görünür ama kapalı
      expect(ready(), isNull);

      await tester.pump(const Duration(seconds: 11));
      expect(
        find.text('Bağlantı yok. Kısa sürede dönmezsen oyundan çıkarılırsın.'),
        findsOneWidget,
      );
    });

    testWidgets('320 dp ve %200 yazı: oyun + bağlantı yok şeridi taşmaz', (
      tester,
    ) async {
      await pump(tester);
      useSmallScreen(tester);
      await gameSnapshot(tester, 'REVEAL');
      expect(tester.takeException(), isNull);
      ws.failNext = 1000;
      ws.last.serverClose(1006);
      await frames(tester, 2);
      await tester.pump(const Duration(seconds: 11));
      expect(
        find.text('Bağlantı yok. Kısa sürede dönmezsen oyundan çıkarılırsın.'),
        findsOneWidget,
      );
      expect(tester.takeException(), isNull);
    });

    testWidgets('Bağlantı yokken kart açılamaz', (tester) async {
      await pump(tester);
      await gameSnapshot(tester, 'REVEAL');
      ActionChip chip() => tester.widget<ActionChip>(
        find.byKey(const ValueKey('reveal-profession')),
      );
      expect(chip().onPressed, isNotNull);
      ws.last.serverClose(1006);
      await frames(tester, 2);
      expect(chip().onPressed, isNull);
    });

    testWidgets('Elenmiş oyuncuya oyundan çıkarılma uyarısı gösterilmez', (
      tester,
    ) async {
      await pump(tester);
      ws.last.serverSend(
        'session.snapshot',
        payload: snapshotPayload(
          roomId: roomId,
          me: 'u-guest',
          status: 'in_game',
          members: gameMembers,
          game: snapshotGame(
            participants: [
              participantJson('u-guest', 'Ayşe', 0, status: 'eliminated'),
              participantJson('u2', 'Mehmet', 1),
              participantJson('u3', 'Zeynep', 2),
            ],
            phase: 'DISCUSSION',
          ),
        ),
      );
      await frames(tester);
      ws.failNext = 1000;
      ws.last.serverClose(1006);
      await frames(tester, 2);
      await tester.pump(const Duration(seconds: 11));
      expect(find.text('Bağlantı yok'), findsOneWidget);
    });

    testWidgets('Kart açma isteği gider', (tester) async {
      backend.on('POST', '/v1/games/g1/actions', (r) async {
        requests.add('action ${jsonEncode(r.data)}');
        return jsonOk({'state_version': 11, 'duplicate': false});
      });
      await pump(tester);
      await gameSnapshot(tester, 'REVEAL');

      await tester.tap(find.byKey(const ValueKey('reveal-profession')));
      await frames(tester);
      await tester.tap(find.text('Aç'));
      await frames(tester);

      final body = jsonDecode(requests.single.substring(7)) as Map;
      expect(body['type'], 'reveal_card');
      expect(body['payload'], {'category': 'profession'});
      expect(body['action_id'], matches(RegExp(r'^[0-9a-f-]{36}$')));
    });

    testWidgets('Oy verilince "Oyunu verdin" görünür', (tester) async {
      backend.on(
        'POST',
        '/v1/games/g1/actions',
        (r) async => jsonOk({'state_version': 11, 'duplicate': false}),
      );
      await pump(tester);
      await gameSnapshot(tester, 'VOTING');

      await tester.tap(find.byKey(const ValueKey('player-u2')));
      await frames(tester);
      await tester.tap(find.text('Oy ver'));
      await frames(tester);

      expect(find.text('Oyunu verdin.'), findsOneWidget);
      expect(find.text('Elemek istediğin oyuncuya dokun.'), findsNothing);
    });

    testWidgets('Sunucu reddederse snackbar; durum değişmez', (tester) async {
      backend.on(
        'POST',
        '/v1/games/g1/actions',
        (r) async => jsonResponse(apiError('INVALID_TARGET'), 409),
      );
      await pump(tester);
      await gameSnapshot(tester, 'VOTING');

      await tester.tap(find.byKey(const ValueKey('player-u2')));
      await frames(tester);
      await tester.tap(find.text('Oy ver'));
      await frames(tester);

      expect(find.text('Bu oyuncu seçilemez.'), findsOneWidget);
      expect(find.text('Oyunu verdin.'), findsNothing);
    });

    testWidgets('Zaten oy verilmişse oy verildi sayılır', (tester) async {
      backend.on(
        'POST',
        '/v1/games/g1/actions',
        (r) async => jsonResponse(apiError('ALREADY_VOTED'), 409),
      );
      await pump(tester);
      await gameSnapshot(tester, 'VOTING');

      await tester.tap(find.byKey(const ValueKey('player-u2')));
      await frames(tester);
      await tester.tap(find.text('Oy ver'));
      await frames(tester);

      expect(find.text('Bu turda oyunu zaten verdin.'), findsOneWidget);
      expect(find.text('Oyunu verdin.'), findsOneWidget);
    });
  });

  group('maç sonu', () {
    final members = [
      wsMember('u-guest', 'Ayşe'),
      wsMember('u2', 'Mehmet'),
      wsMember('u3', 'Zeynep'),
    ];
    final participants = [
      participantJson('u-guest', 'Ayşe', 0),
      participantJson('u2', 'Mehmet', 1),
      participantJson('u3', 'Zeynep', 2, status: 'eliminated'),
    ];

    Map<String, Object?> snap(String status, {String phase = 'VOTING'}) {
      final payload = snapshotPayload(
        roomId: roomId,
        me: 'u-guest',
        status: status,
        members: members,
        game: snapshotGame(participants: participants, phase: phase),
      );
      (payload['room']! as Map<String, Object?>)['current_game_id'] = 'g1';
      return payload;
    }

    testWidgets('game.finished → sonuç; tekrar oyna → lobi', (tester) async {
      backend.on('POST', '/v1/rooms/$roomId/rematch', (r) async {
        requests.add('rematch');
        return jsonOk(
          snapshotPayload(
            roomId: roomId,
            me: 'u-guest',
            version: 9,
            members: members,
          )['room'],
        );
      });
      await pump(tester);
      ws.last.serverSend('session.snapshot', payload: snap('in_game'));
      await frames(tester);

      ws.last.serverSend(
        'game.finished',
        stateVersion: 11,
        payload: resultJson(),
      );
      await frames(tester);

      expect(find.text('Sığınağa girenler'), findsOneWidget);
      expect(backend.calls.where((c) => c.contains('/result')), isEmpty);

      await tester.tap(find.text('Tekrar oyna'));
      await frames(tester);
      expect(requests, ['rematch']);
      expect(find.text('K7M2QX'), findsOneWidget); // lobi
      expect(find.text('Sığınağa girenler'), findsNothing);
    });

    testWidgets('Biten odaya bağlanınca sonuç sunucudan yüklenir', (
      tester,
    ) async {
      backend.on(
        'GET',
        '/v1/games/g1/result',
        (r) async => jsonOk(resultJson()),
      );
      await pump(tester);
      ws.last.serverSend(
        'session.snapshot',
        payload: snap('finished', phase: 'FINISHED'),
      );
      await frames(tester);

      expect(backend.calls, contains('GET /v1/games/g1/result'));
      expect(find.text('Dışarıda kalanlar'), findsOneWidget);
    });

    testWidgets('Sonuç yüklenemezse tekrar denenir', (tester) async {
      var fail = true;
      backend.on('GET', '/v1/games/g1/result', (r) async {
        if (fail) return jsonResponse(apiError('INTERNAL'), 500);
        return jsonOk(resultJson());
      });
      await pump(tester);
      ws.last.serverSend(
        'session.snapshot',
        payload: snap('finished', phase: 'FINISHED'),
      );
      await frames(tester);
      expect(
        find.text('Sunucuda bir sorun oluştu. Biraz sonra tekrar dene.'),
        findsOneWidget,
      );

      fail = false;
      await tester.tap(find.text('Tekrar dene'));
      await frames(tester);
      expect(find.text('Sığınağa girenler'), findsOneWidget);
    });

    testWidgets('Ana ekrana dön → odadan ayrılır', (tester) async {
      await pump(tester);
      ws.last.serverSend('session.snapshot', payload: snap('in_game'));
      await frames(tester);
      ws.last.serverSend(
        'game.finished',
        stateVersion: 11,
        payload: resultJson(),
      );
      await frames(tester);

      await tester.tap(find.text('Ana ekrana dön'));
      await frames(tester);

      expect(requests, ['leave null']);
      expect(exits, 1);
    });
  });

  group('sohbet', () {
    testWidgets('Okunmamış rozeti; panel açılınca sıfırlanır; mesaj gider', (
      tester,
    ) async {
      backend.on('POST', '/v1/rooms/$roomId/messages', (r) async {
        requests.add('message ${jsonEncode(r.data)}');
        return jsonResponse({
          'id': 5,
          'created_at': '2026-11-04T10:05:00Z',
        }, 201);
      });
      await pump(tester);
      await snapshot(tester);
      expect(find.byTooltip('Sohbet'), findsOneWidget);

      ws.last
        ..serverSend(
          'chat.message',
          payload: chatJson(1, 'u2', 'Mehmet', 'selam'),
        )
        ..serverSend(
          'chat.message',
          payload: chatJson(2, 'u2', 'Mehmet', 'orada mısın'),
        );
      await frames(tester);
      expect(find.byTooltip('Sohbet, 2 okunmamış mesaj'), findsOneWidget);

      await tester.tap(find.byTooltip('Sohbet, 2 okunmamış mesaj'));
      await frames(tester);
      expect(find.text('orada mısın'), findsOneWidget);

      await tester.enterText(find.byType(TextField), 'buradayım');
      await tester.pump();
      await tester.tap(find.byTooltip('Gönder'));
      await frames(tester);
      expect(requests.single, contains('"text":"buradayım"'));
      expect(find.text('buradayım'), findsOneWidget);

      await tester.tapAt(const Offset(20, 20)); // paneli kapat
      await frames(tester);
      expect(find.byTooltip('Sohbet'), findsOneWidget);
    });
  });

  group('oyuncu menüsü', () {
    testWidgets('Lobide üyeye dokun → engelle → mesajları gizlenir', (
      tester,
    ) async {
      backend.on(
        'POST',
        '/v1/blocks',
        (r) async => ResponseBodyNoContent.value,
      );
      await pump(tester);
      await snapshot(tester);
      ws.last.serverSend(
        'chat.message',
        payload: chatJson(1, 'u2', 'Mehmet', 'rahatsız edici'),
      );
      await frames(tester);

      // Kendime dokunmak menü açmaz.
      await tester.tap(find.text('Ayşe (sen)'));
      await frames(tester);
      expect(find.text('Raporla'), findsNothing);

      await tester.tap(find.text('Mehmet'));
      await frames(tester);
      await tester.tap(find.text('Engelle'));
      await frames(tester);
      await tester.tap(find.widgetWithText(TextButton, 'Engelle'));
      await frames(tester);
      expect(backend.calls, contains('POST /v1/blocks'));

      await tester.tap(find.byTooltip('Sohbet'));
      await frames(tester);
      expect(find.text('rahatsız edici'), findsNothing);
    });

    testWidgets('Oyunda oyuncu kutusuna dokun → rapor bu odayla gider', (
      tester,
    ) async {
      String? reportBody;
      backend.on('POST', '/v1/reports', (r) async {
        reportBody = jsonEncode(r.data);
        return jsonResponse({'id': 'x'}, 201);
      });
      await pump(tester);
      ws.last.serverSend(
        'session.snapshot',
        payload: snapshotPayload(
          roomId: roomId,
          me: 'u-guest',
          status: 'in_game',
          members: [wsMember('u-guest', 'Ayşe'), wsMember('u2', 'Mehmet')],
          game: snapshotGame(
            participants: [
              participantJson('u-guest', 'Ayşe', 0),
              participantJson('u2', 'Mehmet', 1),
            ],
            phase: 'DISCUSSION',
          ),
        ),
      );
      await frames(tester);

      await tester.tap(find.byKey(const ValueKey('player-u2')));
      await frames(tester);
      await tester.tap(find.text('Raporla'));
      await frames(tester);
      await tester.tap(find.byKey(const ValueKey('reason-cheating')));
      await frames(tester);
      await tester.tap(find.text('Gönder'));
      await frames(tester);

      expect(jsonDecode(reportBody!), {
        'reported_user_id': 'u2',
        'room_id': roomId,
        'reason': 'cheating',
      });
    });
  });

  testWidgets('arka planda 30 sn kalınca WS kapanır, öne gelince yeniden '
      'bağlanır (A13)', (tester) async {
    await pump(tester);
    await snapshot(tester);
    final binding = tester.binding;

    for (final s in [AppLifecycleState.inactive, AppLifecycleState.hidden]) {
      binding.handleAppLifecycleStateChanged(s);
    }
    await tester.pump(const Duration(seconds: 29));
    expect(ws.last.closedByClient, isFalse);

    binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
    await tester.pump(const Duration(seconds: 2));
    expect(ws.last.closedByClient, isTrue);
    expect(ws.last.closeCode, 1000);
    expect(ws.sockets, hasLength(1));

    for (final s in [
      AppLifecycleState.hidden,
      AppLifecycleState.inactive,
      AppLifecycleState.resumed,
    ]) {
      binding.handleAppLifecycleStateChanged(s);
    }
    await frames(tester);
    expect(ws.sockets, hasLength(2));
    await snapshot(tester);
    expect(find.text('Mehmet'), findsOneWidget);
  });
}
