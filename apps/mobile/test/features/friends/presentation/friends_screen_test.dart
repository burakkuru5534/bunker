import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/features/friends/friends.dart';

import '../../../helpers/fake_backend.dart';
import '../../../helpers/fake_http_adapter.dart';
import '../../../helpers/pump_app.dart';
import '../../../helpers/small_screen.dart';

Map<String, Object?> person(String id, String name) => {
  'user_id': id,
  'nickname': name,
};

void main() {
  late FakeBackend backend;
  late Map<String, Object?> friends;

  setUp(() {
    backend = FakeBackend();
    friends = {
      'friends': [person('u-ali', 'Ali')],
      'incoming': [person('u-can', 'Can')],
      'outgoing': [person('u-ece', 'Ece')],
    };
    backend.on('GET', '/v1/friends', (r) async => jsonOk(friends));
  });

  Future<void> pump(WidgetTester tester) async {
    await pumpWithProviders(
      tester,
      const FriendsScreen(),
      overrides: [apiHttpAdapterProvider.overrideWithValue(backend.adapter)],
    );
    await tester.pumpAndSettle();
  }

  Future<void> addName(WidgetTester tester, String name) async {
    await tester.enterText(find.byKey(const ValueKey('friend-name')), name);
    await tester.tap(find.byKey(const ValueKey('friend-add')));
    await tester.pumpAndSettle();
  }

  testWidgets('Arkadaşlar, gelen ve gönderilen istekler listelenir', (
    tester,
  ) async {
    await pump(tester);

    expect(find.text('Gelen istekler'), findsOneWidget);
    expect(find.text('Can'), findsOneWidget);
    expect(find.text('Arkadaşların'), findsOneWidget);
    expect(find.text('Ali'), findsOneWidget);
    expect(find.text('Gönderilen istekler'), findsOneWidget);
    expect(find.text('Ece'), findsOneWidget);
  });

  testWidgets('Boş listede yönlendirme metni; istek bölümleri gizli', (
    tester,
  ) async {
    friends = {
      'friends': <Object?>[],
      'incoming': <Object?>[],
      'outgoing': <Object?>[],
    };
    await pump(tester);

    expect(find.textContaining('Henüz arkadaşın yok'), findsOneWidget);
    expect(find.text('Gelen istekler'), findsNothing);
    expect(find.text('Gönderilen istekler'), findsNothing);
  });

  testWidgets('Kullanıcı adıyla istek gönderilir; alan temizlenir', (
    tester,
  ) async {
    backend.on(
      'POST',
      '/v1/friends/requests',
      (r) async => jsonResponse({
        'status': 'requested',
        'user': person('u-deniz', 'Deniz'),
      }, 201),
    );
    await pump(tester);

    await addName(tester, '  Deniz ');

    expect(
      backend.adapter.requests.firstWhere((r) => r.method == 'POST').data,
      {'nickname': 'Deniz'},
    );
    expect(find.text('Deniz kullanıcısına istek gönderildi.'), findsOneWidget);
    final field = tester.widget<TextField>(
      find.descendant(
        of: find.byKey(const ValueKey('friend-name')),
        matching: find.byType(TextField),
      ),
    );
    expect(field.controller!.text, isEmpty);
    // Liste tazelenir.
    expect(
      backend.calls.where((c) => c == 'GET /v1/friends').length,
      greaterThanOrEqualTo(2),
    );
  });

  testWidgets('Karşılıklı istek: artık arkadaşsınız mesajı', (tester) async {
    backend.on(
      'POST',
      '/v1/friends/requests',
      (r) async =>
          jsonOk({'status': 'friends', 'user': person('u-ece', 'Ece')}),
    );
    await pump(tester);

    await addName(tester, 'Ece');

    expect(find.text('Ece ile artık arkadaşsınız.'), findsOneWidget);
  });

  testWidgets('Hatalar alanın altında gösterilir', (tester) async {
    await pump(tester);

    final cases = <(Map<String, Object?>, int, String)>[
      (apiError('NOT_FOUND'), 404, 'Bu adda bir oyuncu bulunamadı.'),
      (
        apiError(
          'VALIDATION_FAILED',
          details: {
            'fields': {'nickname': 'not_allowed'},
          },
        ),
        422,
        'Kendini arkadaş olarak ekleyemezsin.',
      ),
      (apiError('ALREADY_FRIENDS'), 409, 'Zaten arkadaşsınız.'),
      (
        apiError('FRIEND_LIMIT'),
        409,
        'Sınıra ulaştın. Bazı istekleri veya arkadaşları kaldırıp tekrar dene.',
      ),
    ];
    for (final (body, status, message) in cases) {
      backend.on(
        'POST',
        '/v1/friends/requests',
        (r) async => jsonResponse(body, status),
      );
      await addName(tester, 'Biri');
      expect(find.text(message), findsOneWidget, reason: message);
    }

    // Yazmaya başlayınca hata kalkar.
    await tester.enterText(find.byKey(const ValueKey('friend-name')), 'X');
    await tester.pump();
    expect(find.text(cases.last.$3), findsNothing);
  });

  testWidgets('Boş ad gönderilmez', (tester) async {
    await pump(tester);
    await addName(tester, '   ');
    expect(backend.calls, isNot(contains('POST /v1/friends/requests')));
  });

  testWidgets('Gelen istek kabul edilir / reddedilir; giden geri çekilir', (
    tester,
  ) async {
    backend
      ..on(
        'POST',
        '/v1/friends/requests/u-can/accept',
        (r) async => ResponseBodyNoContent.value,
      )
      ..on(
        'DELETE',
        '/v1/friends/requests/u-can',
        (r) async => ResponseBodyNoContent.value,
      )
      ..on(
        'DELETE',
        '/v1/friends/requests/u-ece',
        (r) async => ResponseBodyNoContent.value,
      );
    await pump(tester);

    await tester.tap(find.text('Kabul et'));
    await tester.pumpAndSettle();
    expect(backend.calls, contains('POST /v1/friends/requests/u-can/accept'));

    await tester.tap(find.text('Reddet'));
    await tester.pumpAndSettle();
    expect(backend.calls, contains('DELETE /v1/friends/requests/u-can'));

    await tester.tap(find.text('Geri çek'));
    await tester.pumpAndSettle();
    expect(backend.calls, contains('DELETE /v1/friends/requests/u-ece'));
  });

  testWidgets('Arkadaşlıktan çıkarma onay ister', (tester) async {
    backend.on(
      'DELETE',
      '/v1/friends/u-ali',
      (r) async => ResponseBodyNoContent.value,
    );
    await pump(tester);

    await tester.tap(find.byTooltip('Arkadaşlıktan çıkar'));
    await tester.pumpAndSettle();
    expect(find.text('Ali arkadaşlıktan çıkarılsın mı?'), findsOneWidget);
    await tester.tap(find.text('Vazgeç'));
    await tester.pumpAndSettle();
    expect(backend.calls, isNot(contains('DELETE /v1/friends/u-ali')));

    await tester.tap(find.byTooltip('Arkadaşlıktan çıkar'));
    await tester.pumpAndSettle();
    await tester.tap(
      find.descendant(
        of: find.byType(AlertDialog),
        matching: find.text('Arkadaşlıktan çıkar'),
      ),
    );
    await tester.pumpAndSettle();
    expect(backend.calls, contains('DELETE /v1/friends/u-ali'));
  });

  testWidgets('Eylem hatası snackbar ile gösterilir', (tester) async {
    backend.on(
      'POST',
      '/v1/friends/requests/u-can/accept',
      (r) async => jsonResponse(apiError('FRIEND_LIMIT'), 409),
    );
    await pump(tester);

    await tester.tap(find.text('Kabul et'));
    await tester.pumpAndSettle();
    expect(find.byType(SnackBar), findsOneWidget);
    expect(find.textContaining('Sınıra ulaştın'), findsOneWidget);
  });

  testWidgets('Liste yüklenemezse hata ve tekrar dene', (tester) async {
    backend.on(
      'GET',
      '/v1/friends',
      (r) async => jsonResponse(apiError('INTERNAL'), 500),
    );
    await pump(tester);
    expect(find.text('Arkadaş listesi yüklenemedi.'), findsOneWidget);

    backend.on('GET', '/v1/friends', (r) async => jsonOk(friends));
    await tester.tap(find.text('Tekrar dene'));
    await tester.pumpAndSettle();
    expect(find.text('Ali'), findsOneWidget);
  });

  testWidgets('320 dp ve %200 yazı: taşma yok', (tester) async {
    useSmallScreen(tester);
    await pump(tester);
    expect(tester.takeException(), isNull);
  });
}
