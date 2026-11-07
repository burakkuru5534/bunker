import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/session_store.dart';
import 'package:siginak/core/ui/brand_logo.dart';
import 'package:siginak/features/home/presentation/home_screen.dart';
import 'package:siginak/features/session/session.dart';

import '../../../helpers/fake_backend.dart';
import '../../../helpers/fake_http_adapter.dart';
import '../../../helpers/pump_app.dart';
import '../../../helpers/small_screen.dart';

void main() {
  late FakeBackend backend;
  late List<String> opened;
  setUp(() {
    backend = FakeBackend();
    opened = [];
  });

  Future<ProviderContainer> pump(
    WidgetTester tester, {
    VoidCallback? onJoin,
    VoidCallback? onOpenSettings,
    AppConfig? config,
  }) async {
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
      HomeScreen(
        onJoin: onJoin ?? () {},
        onOpenSettings: onOpenSettings ?? () {},
        onOpenRoom: opened.add,
      ),
      config: config,
      secureStore: store,
      overrides: [apiHttpAdapterProvider.overrideWithValue(backend.adapter)],
    );
    await tester.pumpAndSettle();
    return ProviderScope.containerOf(tester.element(find.byType(HomeScreen)));
  }

  testWidgets('Logo, selamlama ve local ortam etiketi görünür', (tester) async {
    await pump(tester);

    final logo = tester.widget<Image>(
      find.descendant(of: find.byType(BrandLogo), matching: find.byType(Image)),
    );
    expect((logo.image as AssetImage).assetName, logoAsset);
    expect(logo.semanticLabel, 'BUNKER — Son Kalanlar');
    expect(find.text('SIĞINAK'), findsNothing);
    expect(find.text('Merhaba, Ayşe'), findsOneWidget);
    expect(find.textContaining('Ortam: local'), findsOneWidget);
  });

  test('ortam etiketi yalnızca debug derlemede ve üretim dışında', () {
    AppConfig config(String env) => AppConfig.fromValues(
      environment: env,
      apiBaseUrl: env == 'local'
          ? 'http://10.0.2.2:8080'
          : 'https://api.example.com',
      wsBaseUrl: env == 'local'
          ? 'ws://10.0.2.2:8080'
          : 'wss://api.example.com',
    );
    expect(showEnvironmentLabel(config('local'), debugBuild: true), isTrue);
    expect(showEnvironmentLabel(config('test'), debugBuild: true), isTrue);
    expect(showEnvironmentLabel(config('prod'), debugBuild: true), isFalse);
    // Release derlemesi (mağaza ekran görüntüleri, kapalı test): hiç görünmez.
    expect(showEnvironmentLabel(config('test'), debugBuild: false), isFalse);
    expect(showEnvironmentLabel(config('local'), debugBuild: false), isFalse);
  });

  testWidgets('Production ortamında ortam etiketi gösterilmez', (tester) async {
    final prod = AppConfig.fromValues(
      environment: 'prod',
      apiBaseUrl: 'https://api.example.com',
      wsBaseUrl: 'wss://api.example.com',
    );
    await pump(tester, config: prod);

    expect(find.textContaining('Ortam:'), findsNothing);
  });

  testWidgets('Koda katıl ve ayarlar geri çağrıları tetiklenir', (
    tester,
  ) async {
    var joined = 0;
    var settings = 0;
    await pump(
      tester,
      onJoin: () => joined++,
      onOpenSettings: () => settings++,
    );

    await tester.tap(find.text('Koda katıl'));
    await tester.tap(find.byTooltip('Ayarlar'));

    expect(joined, 1);
    expect(settings, 1);
  });

  testWidgets('Oda oluştur → oda açılır, aktif oda güncellenir', (
    tester,
  ) async {
    final c = await pump(tester);

    await tester.tap(find.text('Oda oluştur'));
    await tester.pumpAndSettle();

    expect(backend.calls.last, 'POST /v1/rooms');
    expect(opened, [FakeBackend.createdRoomId]);
    expect(
      c.read(sessionControllerProvider).activeRoomId,
      FakeBackend.createdRoomId,
    );
  });

  testWidgets('Ağ hatasında snackbar; oda açılmaz, tekrar denenebilir', (
    tester,
  ) async {
    await pump(tester);
    backend.on('POST', '/v1/rooms', networkFailure);

    await tester.tap(find.text('Oda oluştur'));
    await tester.pumpAndSettle();

    expect(
      find.text('İnternet bağlantını kontrol edip tekrar dene.'),
      findsOneWidget,
    );
    expect(opened, isEmpty);

    backend.restore('POST', '/v1/rooms');
    await tester.tap(find.text('Oda oluştur'));
    await tester.pumpAndSettle();
    expect(opened, [FakeBackend.createdRoomId]);
  });

  testWidgets('Zaten odadaysa uyarı ve "Odana dön"', (tester) async {
    const existing = '33333333-3333-4333-8333-333333333333';
    backend.on(
      'POST',
      '/v1/rooms',
      (r) async => jsonResponse(
        apiError('ALREADY_IN_ROOM', details: {'room_id': existing}),
        409,
      ),
    );
    await pump(tester);

    await tester.tap(find.text('Oda oluştur'));
    await tester.pumpAndSettle();

    expect(
      find.text('Zaten başka bir odadasın. Önce oradan ayrıl.'),
      findsOneWidget,
    );
    expect(opened, isEmpty);

    await tester.tap(find.text('Odana dön'));
    await tester.pump();
    expect(opened, [existing]);
  });

  testWidgets('Oluşturma sürerken ikinci basış istek göndermez', (
    tester,
  ) async {
    final gate = Completer<void>();
    backend.on('POST', '/v1/rooms', (r) async {
      await gate.future;
      return jsonResponse(apiError('INTERNAL'), 500);
    });
    await pump(tester);

    await tester.tap(find.text('Oda oluştur'));
    for (var i = 0; i < 5; i++) {
      await tester.pump(const Duration(milliseconds: 10));
    }
    await tester.tap(find.byType(FilledButton));
    await tester.pump();

    expect(backend.calls.where((c) => c == 'POST /v1/rooms'), hasLength(1));
    gate.complete();
    await tester.pumpAndSettle();
    expect(
      find.text('Sunucuda bir sorun oluştu. Biraz sonra tekrar dene.'),
      findsOneWidget,
    );
    expect(opened, isEmpty);
  });

  testWidgets('"Nasıl oynanır?" açılır', (tester) async {
    await pump(tester);
    expect(find.textContaining('Herkese gizli 5 kart'), findsNothing);

    await tester.tap(find.text('Nasıl oynanır?'));
    await tester.pumpAndSettle();

    expect(find.textContaining('Herkese gizli 5 kart'), findsOneWidget);
  });

  testWidgets('320 dp ve %200 yazı: taşma yok; dokunma hedefleri', (
    tester,
  ) async {
    useSmallScreen(tester);
    await pump(tester);
    expect(tester.takeException(), isNull);
    useSmallScreen(tester, size: const Size(360, 640), textScale: 1);
    await tester.pumpAndSettle();
    await expectAccessibleTapTargets(tester);
  });
}
