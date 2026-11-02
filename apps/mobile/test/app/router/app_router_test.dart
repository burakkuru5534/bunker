import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';

import 'package:siginak/app/app.dart';
import 'package:siginak/app/router/app_router.dart';
import 'package:siginak/app/router/app_routes.dart';
import 'package:siginak/core/network/update_required_provider.dart';
import 'package:siginak/core/ui/not_found_screen.dart';
import 'package:siginak/features/home/presentation/home_screen.dart';
import 'package:siginak/features/home/presentation/join_screen.dart';
import 'package:siginak/features/moderation/presentation/blocked_users_screen.dart';
import 'package:siginak/features/room/presentation/room_screen.dart';
import 'package:siginak/features/session/session.dart';
import 'package:siginak/features/settings/presentation/settings_screen.dart';
import 'package:siginak/features/startup/presentation/update_required_screen.dart';

import '../../helpers/pump_app.dart';

void main() {
  const roomId = '0f8fad5b-d9cb-469f-a165-70867728950e';

  Future<ProviderContainer> startApp(WidgetTester tester) async {
    await pumpWithProviders(
      tester,
      const SiginakApp(),
      wrapInMaterialApp: false,
    );
    await tester.pumpAndSettle();
    return ProviderScope.containerOf(tester.element(find.byType(SiginakApp)));
  }

  Future<void> signIn(WidgetTester tester) async {
    await tester.tap(find.text('Başla (geçici)'));
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

    await tester.pageBack();
    await tester.pumpAndSettle();
    expect(find.byType(HomeScreen), findsOneWidget);

    await tester.tap(find.byTooltip('Ayarlar'));
    await tester.pumpAndSettle();
    expect(find.byType(SettingsScreen), findsOneWidget);

    await tester.tap(find.text('Engellenen oyuncular'));
    await tester.pumpAndSettle();
    expect(find.byType(BlockedUsersScreen), findsOneWidget);
  });

  testWidgets('Geçerli oda kimliği oda ekranını açar', (tester) async {
    final c = await startApp(tester);
    await signIn(tester);

    router(c).go(AppRoutes.room(roomId));
    await tester.pumpAndSettle();

    expect(find.byType(RoomScreen), findsOneWidget);
    expect(find.text('Oda: $roomId'), findsOneWidget);
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
}
