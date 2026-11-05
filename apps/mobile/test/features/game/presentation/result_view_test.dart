import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/features/game/game.dart';

import '../../../helpers/fake_ws.dart';
import '../../../helpers/pump_app.dart';
import '../../../helpers/small_screen.dart';

void main() {
  late List<String> calls;
  setUp(() => calls = []);

  Future<void> pump(
    WidgetTester tester, {
    bool isHost = true,
    String endReason = 'capacity_reached',
  }) async {
    tester.view
      ..physicalSize = const Size(1236, 2745)
      ..devicePixelRatio = 3;
    addTearDown(tester.view.reset);
    await pumpWithProviders(
      tester,
      Scaffold(
        body: ResultView(
          result: GameResult.fromJson(resultJson(endReason: endReason)),
          myUserId: 'u-guest',
          isHost: isHost,
          onRematch: () async => calls.add('rematch'),
          onGoHome: () async => calls.add('home'),
        ),
      ),
    );
    await tester.pumpAndSettle();
  }

  testWidgets('Kazananlar, kaybedenler ve tur özeti', (tester) async {
    await pump(tester);

    expect(find.text('Sığınağa girenler'), findsOneWidget);
    expect(find.text('Dışarıda kalanlar'), findsOneWidget);
    expect(find.text('Ayşe (sen)'), findsOneWidget);
    final winners = tester.getTopLeft(find.text('Sığınağa girenler')).dy;
    final losers = tester.getTopLeft(find.text('Dışarıda kalanlar')).dy;
    expect(
      tester.getTopLeft(find.text('Mehmet')).dy,
      inExclusiveRange(winners, losers),
    );
    expect(tester.getTopLeft(find.text('Zeynep')).dy, greaterThan(losers));
    expect(find.text('Tur 1: Zeynep (oylama, 2 oy)'), findsOneWidget);
    expect(find.textContaining('erken bitti'), findsNothing);
  });

  testWidgets('Oyuncu açılınca tüm kartları görünür', (tester) async {
    await pump(tester);
    expect(find.text('Gizli günlük'), findsNothing);

    await tester.tap(find.text('Ayşe (sen)'));
    await tester.pumpAndSettle();

    expect(find.text('Gizli günlük'), findsOneWidget);
  });

  testWidgets('Sahip: tekrar oyna ve ana ekran', (tester) async {
    await pump(tester);
    await tester.tap(find.text('Tekrar oyna'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Ana ekrana dön'));
    await tester.pumpAndSettle();
    expect(calls, ['rematch', 'home']);
  });

  testWidgets('Sahip olmayan tekrar oyna göremez', (tester) async {
    await pump(tester, isHost: false);
    expect(find.text('Tekrar oyna'), findsNothing);
    expect(find.text('Oda sahibi yeni bir oyun başlatabilir.'), findsOneWidget);
  });

  testWidgets('Erken biten oyun notu', (tester) async {
    await pump(tester, endReason: 'abandoned');
    expect(
      find.text('Oyun erken bitti: oyunda yeterli oyuncu kalmadı.'),
      findsOneWidget,
    );
  });

  testWidgets('320 dp ve %200 yazı: taşma yok; dokunma hedefleri', (
    tester,
  ) async {
    await pump(tester);
    useSmallScreen(tester);
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    useSmallScreen(tester, size: const Size(360, 640), textScale: 1);
    await tester.pumpAndSettle();
    await expectAccessibleTapTargets(tester);
  });
}
