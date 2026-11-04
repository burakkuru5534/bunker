import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/features/game/game.dart';

import '../../../helpers/fake_ws.dart';
import '../../../helpers/pump_app.dart';

void main() {
  final players = [
    participantJson(
      'u1',
      'Ayşe',
      0,
      revealed: {'profession': 'Acil tıp doktoru'},
    ),
    participantJson('u2', 'Mehmet', 1, connected: false),
    participantJson(
      'u3',
      'Zeynep',
      2,
      status: 'eliminated',
      revealed: {'profession': 'Aşçı', 'secret': 'Sır'},
    ),
    participantJson('u4', 'Uzunisimlioyuncu', 3),
  ];

  late DateTime now;
  setUp(() => now = DateTime.utc(2026, 11, 4, 10));

  GameView game({
    String phase = 'REVEAL',
    String? turn = 'u2',
    int readyCount = 0,
    Map<String, Object?>? votes,
    Map<String, Object?>? last,
  }) => GameView.fromSnapshot(
    snapshotGame(
      participants: players,
      phase: phase,
      turnUserId: turn,
      deadline: now.add(const Duration(seconds: 75)).toIso8601String(),
      myCards: myCardsJson(revealed: {'profession'}),
      readyCount: readyCount,
      voteProgress: votes,
      lastRoundResult: last,
    ),
    serverTime: now,
    receivedAt: now,
  );

  Future<void> pump(WidgetTester tester, GameView g, {String me = 'u1'}) async {
    await pumpWithProviders(
      tester,
      Scaffold(
        body: GameBody(game: g, myUserId: me, clock: () => now),
      ),
    );
    await tester.pump();
  }

  testWidgets('Üst çubuk: senaryo, tur, aşama, geri sayım', (tester) async {
    await pump(tester, game());

    expect(find.text('Göktaşı'), findsOneWidget);
    expect(find.text('Tur 1/2'), findsOneWidget);
    expect(find.text('KART AÇMA'), findsOneWidget);
    expect(find.text('01:15'), findsOneWidget);

    now = now.add(const Duration(seconds: 2));
    await tester.pump(const Duration(seconds: 1));
    expect(find.text('01:13'), findsOneWidget);

    now = now.add(const Duration(minutes: 5));
    await tester.pump(const Duration(seconds: 1));
    expect(find.text('00:00'), findsOneWidget);
  });

  testWidgets('Izgara: oyuncular, açılmış kartlar, durumlar', (tester) async {
    await pump(tester, game());

    expect(find.text('Ayşe (sen)'), findsOneWidget);
    expect(find.text('Acil tıp doktoru'), findsOneWidget);
    expect(find.text('Sırada'), findsOneWidget); // u2'nin sırası
    expect(find.text('Elendi'), findsOneWidget);
    expect(find.text('Sır'), findsOneWidget); // elenenin kartları açık
    expect(find.text('Mehmet kartını açıyor…'), findsOneWidget);
  });

  testWidgets('Sıra bendeyse bilgi değişir', (tester) async {
    await pump(tester, game(turn: 'u1'));
    expect(find.text('Sıra sende. Bir kartını aç.'), findsOneWidget);
  });

  testWidgets('Aşama bilgileri: tartışma, oylama, eşitlik, eleme', (
    tester,
  ) async {
    await pump(tester, game(phase: 'DISCUSSION', readyCount: 1));
    expect(find.text('İkna etme zamanı.'), findsOneWidget);
    expect(find.text('1/3 hazır'), findsOneWidget);

    await pump(tester, game(phase: 'VOTING', votes: {'cast': 2, 'total': 3}));
    expect(find.text('2/3 oy verildi'), findsOneWidget);

    await pump(tester, game(phase: 'TIE_BREAK'));
    expect(find.text('Eşitlik! İzleyiciler karar veriyor…'), findsOneWidget);

    await pump(
      tester,
      game(
        phase: 'ELIMINATION',
        last: {
          'round': 1,
          'user_id': 'u3',
          'method': 'vote',
          'tally': <Object?>[],
          'lottery_candidates': <Object?>[],
          'cards': <Object?>[],
        },
      ),
    );
    expect(find.text('Zeynep sığınağın dışında kaldı.'), findsOneWidget);
  });

  testWidgets('Elenen oyuncu izleyici şeridini görür', (tester) async {
    await pump(tester, game(), me: 'u3');
    expect(
      find.text('İzleyicisin. Eşitlik olursa kararı izleyiciler verir.'),
      findsOneWidget,
    );

    await pump(tester, game());
    expect(find.textContaining('İzleyicisin'), findsNothing);
  });

  testWidgets('Senaryo tanıtımı ve senaryo penceresi', (tester) async {
    await pump(tester, game(phase: 'SCENARIO_INTRO', turn: null));
    expect(find.text('Sığınakta neler var?'), findsOneWidget);
    expect(find.text('Teknik kütüphane'), findsOneWidget);
    expect(find.text('Sığınağa yalnızca 2 kişi girebilir.'), findsOneWidget);

    await pump(tester, game());
    expect(find.text('Sığınakta neler var?'), findsNothing);
    await tester.tap(find.text('Göktaşı'));
    await tester.pumpAndSettle();
    expect(find.text('Sığınakta neler var?'), findsOneWidget);
  });

  testWidgets('Kartlarım: 5 kart, açık/kapalı', (tester) async {
    await pump(tester, game());

    await tester.tap(find.text('Kartlarım'));
    await tester.pumpAndSettle();

    expect(find.byKey(const ValueKey('my-card-profession')), findsOneWidget);
    expect(find.text('Astım'), findsOneWidget);
    expect(find.text('Açık'), findsOneWidget);
    expect(find.text('Kapalı'), findsNWidgets(4));
  });

  testWidgets('Küçük ekranda taşma yok (360×640)', (tester) async {
    tester.view
      ..physicalSize = const Size(1080, 1920)
      ..devicePixelRatio = 3;
    addTearDown(tester.view.reset);

    for (final phase in ['SCENARIO_INTRO', 'REVEAL', 'DISCUSSION', 'VOTING']) {
      await pump(tester, game(phase: phase));
      expect(tester.takeException(), isNull, reason: phase);
    }
  });
}
