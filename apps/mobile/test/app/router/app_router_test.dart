import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import 'package:siginak/app/app.dart';
import 'package:siginak/app/router/app_router.dart';
import 'package:siginak/app/router/app_routes.dart';
import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/network/update_required_provider.dart';
import 'package:siginak/core/realtime/realtime_client_provider.dart';
import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/session_store.dart';
import 'package:siginak/core/ui/not_found_screen.dart';
import 'package:siginak/features/home/presentation/home_screen.dart';
import 'package:siginak/features/home/presentation/join_screen.dart';
import 'package:siginak/features/moderation/presentation/blocked_users_screen.dart';
import 'package:siginak/features/room/presentation/room_screen.dart';
import 'package:siginak/features/session/session.dart';
import 'package:siginak/features/settings/presentation/settings_screen.dart';
import 'package:siginak/features/startup/presentation/splash_screen.dart';
import 'package:siginak/features/startup/presentation/update_required_screen.dart';

import '../../helpers/fake_backend.dart';
import '../../helpers/fake_http_adapter.dart';
import '../../helpers/fake_ws.dart';

import '../../helpers/pump_app.dart';

void main() {
  const roomId = '0f8fad5b-d9cb-469f-a165-70867728950e';

  late FakeBackend backend;
  late FakeWsConnector ws;
  setUp(() {
    backend = FakeBackend();
    ws = FakeWsConnector();
  });

  /// Animasyonu süren ekranlarda (oda: bağlanıyor göstergesi) pumpAndSettle
  /// bitmez; sabit sayıda kare ilerletilir.
  Future<void> settle(WidgetTester tester) async {
    for (var i = 0; i < 20; i++) {
      await tester.pump(const Duration(milliseconds: 50));
    }
  }

  /// Oda ekranının bağlandığı oda.
  String? wsRoom() => ws.uris.lastOrNull?.queryParameters['room_id'];

  Future<ProviderContainer> startApp(
    WidgetTester tester, {
    SecureStore? store,
  }) async {
    await pumpWithProviders(
      tester,
      const SiginakApp(),
      wrapInMaterialApp: false,
      secureStore: store,
      overrides: [
        apiHttpAdapterProvider.overrideWithValue(backend.adapter),
        wsConnectorProvider.overrideWithValue(ws.call),
      ],
    );
    await settle(tester);
    return ProviderScope.containerOf(tester.element(find.byType(SiginakApp)));
  }

  Future<void> signIn(WidgetTester tester) async {
    await tester.enterText(find.byType(TextField), 'Ayşe');
    await tester.tap(find.text('Başla'));
    await tester.pumpAndSettle();
  }

  GoRouter router(ProviderContainer c) => c.read(appRouterProvider);

  testWidgets('Oturum açılınca ana ekrana geçilir', (tester) async {
    await startApp(tester);
    expect(find.byType(WelcomeScreen), findsOneWidget);

    await signIn(tester);

    expect(find.byType(HomeScreen), findsOneWidget);
  });

  testWidgets('Ana ekrandan koda katıl ve ayarlar açılır, geri dönülür', (
    tester,
  ) async {
    await startApp(tester);
    await signIn(tester);

    await tester.tap(find.text('Koda katıl'));
    await tester.pumpAndSettle();
    expect(find.byType(JoinScreen), findsOneWidget);
    expect(find.byTooltip('Geri'), findsOneWidget); // Türkçe Material metinleri

    await tester.tap(find.byType(BackButton));
    await tester.pumpAndSettle();
    expect(find.byType(HomeScreen), findsOneWidget);

    await tester.tap(find.byTooltip('Ayarlar'));
    await tester.pumpAndSettle();
    expect(find.byType(SettingsScreen), findsOneWidget);

    await tester.tap(find.text('Engellenen oyuncular'));
    await tester.pumpAndSettle();
    expect(find.byType(BlockedUsersScreen), findsOneWidget);
  });

  testWidgets('Oda oluşturunca oda ekranına geçilir; geri ana ekrana dönmez', (
    tester,
  ) async {
    final c = await startApp(tester);
    await signIn(tester);

    await tester.tap(find.text('Oda oluştur'));
    await settle(tester);

    expect(find.byType(RoomScreen), findsOneWidget);
    expect(wsRoom(), FakeBackend.createdRoomId);
    expect(router(c).canPop(), isFalse);
  });

  testWidgets('Koda katılınca oda ekranına geçilir', (tester) async {
    final c = await startApp(tester);
    await signIn(tester);

    await tester.tap(find.text('Koda katıl'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), 'K7M2QX');
    await settle(tester);

    expect(find.byType(RoomScreen), findsOneWidget);
    expect(wsRoom(), FakeBackend.joinedRoomId);
    expect(router(c).canPop(), isFalse);
  });

  testWidgets('Geçerli oda kimliği oda ekranını açar', (tester) async {
    final c = await startApp(tester);
    await signIn(tester);

    router(c).go(AppRoutes.room(roomId));
    await settle(tester);

    expect(find.byType(RoomScreen), findsOneWidget);
    expect(wsRoom(), roomId);
  });

  testWidgets('Geçersiz oda kimliği ve bilinmeyen adres bulunamadı gösterir', (
    tester,
  ) async {
    final c = await startApp(tester);
    await signIn(tester);

    router(c).go('/room/abc');
    await tester.pumpAndSettle();
    expect(find.byType(NotFoundScreen), findsOneWidget);

    router(c).go('/yok-boyle-bir-sayfa');
    await tester.pumpAndSettle();
    expect(find.byType(NotFoundScreen), findsOneWidget);

    await tester.tap(find.text('Ana ekrana dön'));
    await tester.pumpAndSettle();
    expect(find.byType(HomeScreen), findsOneWidget);
  });

  testWidgets('Oturum yokken korumalı adrese gidilemez', (tester) async {
    final c = await startApp(tester);

    router(c).go(AppRoutes.settings);
    await tester.pumpAndSettle();

    expect(find.byType(WelcomeScreen), findsOneWidget);
  });

  testWidgets('Güncelleme gerekince /update ekranına kilitlenir', (
    tester,
  ) async {
    final c = await startApp(tester);
    await signIn(tester);

    c.read(updateRequiredProvider.notifier).markRequired();
    await tester.pumpAndSettle();
    expect(find.byType(UpdateRequiredScreen), findsOneWidget);

    router(c).go(AppRoutes.home);
    await tester.pumpAndSettle();
    expect(find.byType(UpdateRequiredScreen), findsOneWidget);
    expect(find.byType(Scaffold), findsOneWidget);
  });

  Future<InMemorySecureStore> storedSession() async {
    final store = InMemorySecureStore();
    await SessionStore(store).save(
      StoredSession(
        token: 't',
        expiresAt: DateTime.now().add(const Duration(days: 30)),
        userId: 'u',
        nickname: 'Ayşe',
      ),
    );
    return store;
  }

  testWidgets('Saklı oturumu olan kullanıcı doğrudan ana ekrana gider', (
    tester,
  ) async {
    await startApp(tester, store: await storedSession());

    expect(find.byType(HomeScreen), findsOneWidget);
    expect(find.byType(WelcomeScreen), findsNothing);
    expect(backend.calls, ['GET /v1/me']);
  });

  testWidgets('Aktif odası olan kullanıcı açılışta odaya döner', (
    tester,
  ) async {
    backend.activeRoomId = roomId;

    await startApp(tester, store: await storedSession());

    expect(find.byType(RoomScreen), findsOneWidget);
    expect(wsRoom(), roomId);
  });

  testWidgets('Oturum doğrulanamazsa açılışta hata ve tekrar dene', (
    tester,
  ) async {
    backend.on('GET', '/v1/me', networkFailure);

    await startApp(tester, store: await storedSession());

    expect(find.byType(SplashScreen), findsOneWidget);
    expect(
      find.text('İnternet bağlantını kontrol edip tekrar dene.'),
      findsOneWidget,
    );

    backend.restore('GET', '/v1/me');
    await tester.tap(find.text('Tekrar dene'));
    await tester.pumpAndSettle();

    expect(find.byType(HomeScreen), findsOneWidget);
  });

  testWidgets('Saklı oturum sunucuda geçersizse hoş geldin ekranı', (
    tester,
  ) async {
    backend.on(
      'GET',
      '/v1/me',
      (r) async => jsonResponse(apiError('UNAUTHORIZED'), 401),
    );

    await startApp(tester, store: await storedSession());

    expect(find.byType(WelcomeScreen), findsOneWidget);
  });

  testWidgets('Oturum kapatılınca hoş geldin ekranına dönülür', (tester) async {
    final c = await startApp(tester);
    await signIn(tester);
    expect(find.byType(HomeScreen), findsOneWidget);

    await c.read(sessionControllerProvider.notifier).signOut();
    await tester.pumpAndSettle();

    expect(find.byType(WelcomeScreen), findsOneWidget);
  });
}
