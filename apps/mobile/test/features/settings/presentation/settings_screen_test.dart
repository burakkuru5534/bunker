import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/session_store.dart';
import 'package:siginak/features/session/session.dart';
import 'package:siginak/features/settings/presentation/settings_screen.dart';

import '../../../helpers/fake_backend.dart';
import '../../../helpers/fake_http_adapter.dart';
import '../../../helpers/pump_app.dart';

void main() {
  late FakeBackend backend;
  late InMemorySecureStore store;
  late int blocksOpened;

  setUp(() async {
    backend = FakeBackend();
    store = InMemorySecureStore();
    blocksOpened = 0;
    await SessionStore(store).save(
      StoredSession(
        token: 't',
        expiresAt: DateTime.now().add(const Duration(days: 30)),
        userId: 'u-guest',
        nickname: 'Ayşe',
      ),
    );
  });

  Future<ProviderContainer> pump(WidgetTester tester) async {
    await pumpWithProviders(
      tester,
      SettingsScreen(onOpenBlockedUsers: () => blocksOpened++),
      secureStore: store,
      overrides: [apiHttpAdapterProvider.overrideWithValue(backend.adapter)],
    );
    await tester.pumpAndSettle();
    return ProviderScope.containerOf(
      tester.element(find.byType(SettingsScreen)),
    );
  }

  Future<void> tapAndSettle(WidgetTester tester, Finder f) async {
    await tester.tap(f);
    await tester.pumpAndSettle();
  }

  testWidgets('Takma ad, engellenenler, sürüm', (tester) async {
    await pump(tester);
    expect(find.text('Ayşe'), findsOneWidget);
    expect(find.text('Sürüm 0.1.0'), findsOneWidget);

    await tapAndSettle(tester, find.text('Engellenen oyuncular'));
    expect(blocksOpened, 1);
  });

  testWidgets('Takma ad değiştirilir', (tester) async {
    await pump(tester);
    await tapAndSettle(tester, find.text('Takma ad'));
    await tester.enterText(
      find.descendant(
        of: find.byType(AlertDialog),
        matching: find.byType(TextField),
      ),
      'Yeni Ad',
    );
    await tapAndSettle(tester, find.text('Kaydet'));

    expect(find.text('Yeni Ad'), findsOneWidget);
    expect(backend.calls, contains('PATCH /v1/me'));
  });

  testWidgets('Hesap silme iki adımlı; ikinci adımda vazgeçilebilir', (
    tester,
  ) async {
    await pump(tester);
    await tapAndSettle(tester, find.byKey(const ValueKey('delete-account')));
    expect(find.text('Hesabın silinsin mi?'), findsOneWidget);
    await tapAndSettle(tester, find.text('Devam'));
    expect(find.text('Emin misin?'), findsOneWidget);
    await tapAndSettle(tester, find.text('Vazgeç'));

    expect(backend.calls, isNot(contains('DELETE /v1/me')));
  });

  testWidgets('Hesap silinince oturum kapanır', (tester) async {
    backend.on('DELETE', '/v1/me', (r) async => ResponseBodyNoContent.value);
    final c = await pump(tester);

    await tapAndSettle(tester, find.byKey(const ValueKey('delete-account')));
    await tapAndSettle(tester, find.text('Devam'));
    await tapAndSettle(
      tester,
      find.descendant(
        of: find.byType(AlertDialog),
        matching: find.text('Hesabımı sil'),
      ),
    );

    expect(backend.calls, contains('DELETE /v1/me'));
    expect(
      c.read(sessionControllerProvider).status,
      SessionStatus.unauthenticated,
    );
    expect(store.values, isEmpty);
  });

  testWidgets('Silme başarısızsa oturum kalır, hata gösterilir', (
    tester,
  ) async {
    backend.on('DELETE', '/v1/me', networkFailure);
    final c = await pump(tester);

    await tapAndSettle(tester, find.byKey(const ValueKey('delete-account')));
    await tapAndSettle(tester, find.text('Devam'));
    await tapAndSettle(
      tester,
      find.descendant(
        of: find.byType(AlertDialog),
        matching: find.text('Hesabımı sil'),
      ),
    );

    expect(
      c.read(sessionControllerProvider).status,
      SessionStatus.authenticated,
    );
    expect(
      find.text('İnternet bağlantını kontrol edip tekrar dene.'),
      findsOneWidget,
    );
  });
}
