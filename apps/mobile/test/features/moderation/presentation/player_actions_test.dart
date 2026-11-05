import 'dart:convert';

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
  late int blocked;
  late List<String> bodies;

  setUp(() {
    backend = FakeBackend();
    blocked = 0;
    bodies = [];
    backend
      ..on('POST', '/v1/reports', (r) async {
        bodies.add(jsonEncode(r.data));
        return jsonResponse({'id': 'rep1'}, 201);
      })
      ..on('POST', '/v1/blocks', (r) async {
        bodies.add(jsonEncode(r.data));
        return ResponseBodyNoContent.value;
      });
  });

  Future<void> open(WidgetTester tester) async {
    await pumpWithProviders(
      tester,
      Scaffold(
        body: Builder(
          builder: (context) => TextButton(
            onPressed: () => showPlayerActions(
              context,
              userId: 'u2',
              nickname: 'Mehmet',
              roomId: 'r1',
              onBlocked: () => blocked++,
            ),
            child: const Text('menü'),
          ),
        ),
      ),
      overrides: [apiHttpAdapterProvider.overrideWithValue(backend.adapter)],
    );
    await tester.tap(find.text('menü'));
    await tester.pumpAndSettle();
  }

  testWidgets('Raporla: neden seçilmeden gönderilemez; not ile gider', (
    tester,
  ) async {
    await open(tester);
    await tester.tap(find.text('Raporla'));
    await tester.pumpAndSettle();
    expect(find.text('Mehmet adlı oyuncuyu raporla'), findsOneWidget);

    TextButton send() =>
        tester.widget<TextButton>(find.widgetWithText(TextButton, 'Gönder'));
    expect(send().onPressed, isNull);

    await tester.tap(find.byKey(const ValueKey('reason-hate')));
    await tester.enterText(find.byType(TextField), 'kötü söz');
    await tester.pump();
    await tester.tap(find.text('Gönder'));
    await tester.pumpAndSettle();

    expect(jsonDecode(bodies.single), {
      'reported_user_id': 'u2',
      'room_id': 'r1',
      'reason': 'hate',
      'note': 'kötü söz',
    });
    expect(find.byType(AlertDialog), findsNothing);
    expect(find.text('Raporun alındı. Teşekkürler.'), findsOneWidget);
  });

  testWidgets('Rapor reddedilirse pencerede hata', (tester) async {
    backend.on(
      'POST',
      '/v1/reports',
      (r) async => jsonResponse(apiError('RATE_LIMITED'), 429),
    );
    await open(tester);
    await tester.tap(find.text('Raporla'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('reason-spam')));
    await tester.pump();
    await tester.tap(find.text('Gönder'));
    await tester.pumpAndSettle();

    expect(find.byType(AlertDialog), findsOneWidget);
    expect(
      find.text('Çok hızlı gidiyorsun. Biraz bekleyip tekrar dene.'),
      findsOneWidget,
    );
  });

  testWidgets('Engelle: onaylı; başarıda geri çağrı', (tester) async {
    await open(tester);
    await tester.tap(find.text('Engelle'));
    await tester.pumpAndSettle();
    expect(find.text('Mehmet engellensin mi?'), findsOneWidget);
    await tester.tap(find.text('Vazgeç'));
    await tester.pumpAndSettle();
    expect(bodies, isEmpty);

    await tester.tap(find.text('menü'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Engelle'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(TextButton, 'Engelle'));
    await tester.pumpAndSettle();

    expect(jsonDecode(bodies.single), {'user_id': 'u2'});
    expect(blocked, 1);
    expect(find.text('Mehmet engellendi.'), findsOneWidget);
  });

  testWidgets('Engelleme başarısızsa geri çağrı yok', (tester) async {
    backend.on('POST', '/v1/blocks', networkFailure);
    await open(tester);
    await tester.tap(find.text('Engelle'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(TextButton, 'Engelle'));
    await tester.pumpAndSettle();

    expect(blocked, 0);
    expect(
      find.text('İnternet bağlantını kontrol edip tekrar dene.'),
      findsOneWidget,
    );
  });

  testWidgets('320 dp ve %200 yazı: menü ve rapor penceresi taşmaz', (
    tester,
  ) async {
    useSmallScreen(tester);
    await open(tester);
    expect(tester.takeException(), isNull);
    await tester.tap(find.text('Raporla'));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  });
}
