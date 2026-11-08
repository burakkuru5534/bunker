import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/session_store.dart';
import 'package:siginak/features/home/presentation/join_screen.dart';
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
    String? initialCode,
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
      JoinScreen(onOpenRoom: opened.add, initialCode: initialCode),
      secureStore: store,
      overrides: [apiHttpAdapterProvider.overrideWithValue(backend.adapter)],
    );
    await tester.pumpAndSettle();
    return ProviderScope.containerOf(tester.element(find.byType(JoinScreen)));
  }

  Finder codeField() => find.byType(TextField);

  FilledButton joinButton(WidgetTester tester) =>
      tester.widget<FilledButton>(find.byType(FilledButton));

  void failJoin(
    String code, {
    Map<String, Object?>? details,
    int status = 409,
  }) {
    backend.on(
      'POST',
      '/v1/rooms/join',
      (r) async => jsonResponse(apiError(code, details: details), status),
    );
  }

  testWidgets('Davetten gelen kodla ekran açılır açılmaz katılır', (
    tester,
  ) async {
    await pump(tester, initialCode: 'k7m2qx');

    final joins = backend.adapter.requests
        .where((r) => r.path == '/v1/rooms/join')
        .toList();
    expect(joins, hasLength(1));
    expect(joins.single.data, {'invite_code': 'K7M2QX'});
    expect(opened, [FakeBackend.joinedRoomId]);
  });

  testWidgets('Davet kodu eksik/bozuksa otomatik katılınmaz', (tester) async {
    await pump(tester, initialCode: 'K7M');
    expect(backend.calls.where((c) => c.contains('join')), isEmpty);
    expect(opened, isEmpty);
  });

  testWidgets('Kod tamamlanmadan Katıl kapalı; tamamlanınca katılır', (
    tester,
  ) async {
    final c = await pump(tester);
    expect(joinButton(tester).onPressed, isNull);

    await tester.enterText(codeField(), 'k7m2q');
    await tester.pump();
    expect(joinButton(tester).onPressed, isNull);
    expect(backend.calls.where((c) => c.contains('join')), isEmpty);

    // Altıncı karakter: otomatik katılma; küçük harf büyütülür.
    await tester.enterText(codeField(), 'k7m2qx');
    await tester.pumpAndSettle();

    final join = backend.adapter.requests.singleWhere(
      (r) => r.path == '/v1/rooms/join',
    );
    expect(join.data, {'invite_code': 'K7M2QX'});
    expect(opened, [FakeBackend.joinedRoomId]);
    expect(
      c.read(sessionControllerProvider).activeRoomId,
      FakeBackend.joinedRoomId,
    );
  });

  testWidgets('Hata kodları satır içinde; kod değişince silinir', (
    tester,
  ) async {
    await pump(tester);
    final cases = {
      'INVITE_CODE_NOT_FOUND': 'Bu koda ait bir oda yok ya da oda kapanmış.',
      'ROOM_FULL': 'Bu oda dolu. Başka bir kod dene.',
      'GAME_IN_PROGRESS': 'Bu odada oyun başlamış. Bitince tekrar dene.',
      'KICKED': 'Bu odadan çıkarıldın, tekrar katılamazsın.',
    };
    for (final MapEntry(key: code, value: text) in cases.entries) {
      failJoin(code);
      await tester.enterText(codeField(), 'K7M2QX');
      await tester.pumpAndSettle();
      expect(find.text(text), findsOneWidget, reason: code);
      expect(opened, isEmpty);

      await tester.enterText(codeField(), 'K7M2Q');
      await tester.pump();
      expect(find.text(text), findsNothing, reason: code);
    }
  });

  testWidgets('Biçimi bozuk kod için özel mesaj', (tester) async {
    failJoin(
      'VALIDATION_FAILED',
      status: 400,
      details: {
        'fields': {'invite_code': 'invalid'},
      },
    );
    await pump(tester);

    await tester.enterText(codeField(), 'K7M2QX');
    await tester.pumpAndSettle();

    expect(
      find.text('Bu kod geçerli değil. Kontrol edip tekrar dene.'),
      findsOneWidget,
    );
  });

  testWidgets('Takma ad çakışması → ad değiştirilir ve yeniden katılınır', (
    tester,
  ) async {
    failJoin('NICKNAME_TAKEN_IN_ROOM');
    final c = await pump(tester);

    await tester.enterText(codeField(), 'K7M2QX');
    await tester.pumpAndSettle();
    expect(
      find.text('Bu odada aynı takma adla biri var. Takma adını değiştir.'),
      findsOneWidget,
    );

    await tester.tap(
      find.widgetWithText(OutlinedButton, 'Takma adını değiştir'),
    );
    await tester.pumpAndSettle();
    expect(find.byType(AlertDialog), findsOneWidget);

    backend.restore('POST', '/v1/rooms/join');
    await tester.enterText(
      find.descendant(
        of: find.byType(AlertDialog),
        matching: find.byType(TextField),
      ),
      'Ayşe 2',
    );
    await tester.tap(find.text('Kaydet'));
    await tester.pumpAndSettle();

    expect(
      backend.calls,
      containsAllInOrder(['PATCH /v1/me', 'POST /v1/rooms/join']),
    );
    expect(c.read(sessionControllerProvider).nickname, 'Ayşe 2');
    expect(opened, [FakeBackend.joinedRoomId]);
  });

  testWidgets(
    'Ad değiştirme penceresi sunucu hatasını gösterir, vazgeçilebilir',
    (tester) async {
      failJoin('NICKNAME_TAKEN_IN_ROOM');
      backend.on(
        'PATCH',
        '/v1/me',
        (r) async => jsonResponse(
          apiError(
            'NICKNAME_INVALID',
            details: {
              'fields': {'nickname': 'profanity'},
            },
          ),
          400,
        ),
      );
      await pump(tester);
      await tester.enterText(codeField(), 'K7M2QX');
      await tester.pumpAndSettle();
      await tester.tap(
        find.widgetWithText(OutlinedButton, 'Takma adını değiştir'),
      );
      await tester.pumpAndSettle();

      final field = find.descendant(
        of: find.byType(AlertDialog),
        matching: find.byType(TextField),
      );
      await tester.enterText(field, 'ab');
      await tester.pump();
      expect(find.text('En az 3 karakter olmalı.'), findsOneWidget);

      await tester.enterText(field, 'Kötü Ad');
      await tester.tap(find.text('Kaydet'));
      await tester.pumpAndSettle();
      expect(
        find.text('Bu takma ad uygun değil, başka bir tane seç.'),
        findsOneWidget,
      );

      await tester.tap(find.text('Vazgeç'));
      await tester.pumpAndSettle();
      expect(find.byType(AlertDialog), findsNothing);
      expect(
        backend.calls.where((c) => c == 'POST /v1/rooms/join'),
        hasLength(1),
      );
    },
  );

  testWidgets('Zaten başka odadaysa "Odana dön"', (tester) async {
    const existing = '33333333-3333-4333-8333-333333333333';
    failJoin('ALREADY_IN_ROOM', details: {'room_id': existing});
    await pump(tester);

    await tester.enterText(codeField(), 'K7M2QX');
    await tester.pumpAndSettle();
    await tester.tap(find.text('Odana dön'));
    await tester.pumpAndSettle();

    expect(opened, [existing]);
  });

  testWidgets('Ağ hatası satır içinde; Katıl ile tekrar denenir', (
    tester,
  ) async {
    backend.on('POST', '/v1/rooms/join', networkFailure);
    await pump(tester);

    await tester.enterText(codeField(), 'K7M2QX');
    await tester.pumpAndSettle();
    expect(
      find.text('İnternet bağlantını kontrol edip tekrar dene.'),
      findsOneWidget,
    );

    backend.restore('POST', '/v1/rooms/join');
    await tester.tap(find.text('Katıl'));
    await tester.pumpAndSettle();
    expect(opened, [FakeBackend.joinedRoomId]);
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
