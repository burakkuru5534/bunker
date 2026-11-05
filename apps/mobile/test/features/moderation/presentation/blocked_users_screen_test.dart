import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/features/moderation/moderation.dart';

import '../../../helpers/fake_backend.dart';
import '../../../helpers/fake_http_adapter.dart';
import '../../../helpers/pump_app.dart';
import '../../../helpers/small_screen.dart';

void main() {
  late FakeBackend backend;
  setUp(() => backend = FakeBackend());

  Future<void> pump(WidgetTester tester) async {
    await pumpWithProviders(
      tester,
      const BlockedUsersScreen(),
      overrides: [apiHttpAdapterProvider.overrideWithValue(backend.adapter)],
    );
    await tester.pumpAndSettle();
  }

  List<Map<String, Object?>> list(List<(String, String)> users) => [
    for (final (id, nick) in users)
      {'user_id': id, 'nickname': nick, 'created_at': '2026-11-04T10:00:00Z'},
  ];

  testWidgets('Liste; engel kaldırılınca düşer', (tester) async {
    backend
      ..on(
        'GET',
        '/v1/blocks',
        (r) async => jsonOk(list([('u2', 'Mehmet'), ('u3', 'Zeynep')])),
      )
      ..on('DELETE', '/v1/blocks/u2', (r) async => ResponseBodyNoContent.value);
    await pump(tester);

    expect(find.text('Mehmet'), findsOneWidget);
    expect(find.text('Zeynep'), findsOneWidget);

    await tester.tap(
      find.descendant(
        of: find.byKey(const ValueKey('blocked-u2')),
        matching: find.text('Engeli kaldır'),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Mehmet'), findsNothing);
    expect(find.text('Mehmet artık engelli değil.'), findsOneWidget);
    expect(backend.calls, contains('DELETE /v1/blocks/u2'));
  });

  testWidgets('Boş liste', (tester) async {
    backend.on('GET', '/v1/blocks', (r) async => jsonOk(<Object?>[]));
    await pump(tester);
    expect(find.text('Engellediğin kimse yok.'), findsOneWidget);
  });

  testWidgets('Yüklenemezse tekrar dene', (tester) async {
    var fail = true;
    backend.on('GET', '/v1/blocks', (r) async {
      if (fail) return jsonResponse(apiError('INTERNAL'), 500);
      return jsonOk(list([('u2', 'Mehmet')]));
    });
    await pump(tester);
    expect(
      find.text('Sunucuda bir sorun oluştu. Biraz sonra tekrar dene.'),
      findsOneWidget,
    );

    fail = false;
    await tester.tap(find.text('Tekrar dene'));
    await tester.pumpAndSettle();
    expect(find.text('Mehmet'), findsOneWidget);
  });

  testWidgets('Engel kaldırılamazsa liste kalır, hata gösterilir', (
    tester,
  ) async {
    backend
      ..on('GET', '/v1/blocks', (r) async => jsonOk(list([('u2', 'Mehmet')])))
      ..on('DELETE', '/v1/blocks/u2', networkFailure);
    await pump(tester);

    await tester.tap(find.text('Engeli kaldır'));
    await tester.pumpAndSettle();

    expect(find.text('Mehmet'), findsOneWidget);
    expect(
      find.text('İnternet bağlantını kontrol edip tekrar dene.'),
      findsOneWidget,
    );
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
