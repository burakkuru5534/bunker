import 'dart:async';

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
    int round = 1,
    int readyCount = 0,
    Map<String, Object?>? votes,
    Map<String, Object?>? last,
    Map<String, Object?>? state,
    Map<String, Object?>? tiebreak,
    Set<String> revealed = const {'profession'},
  }) => GameView.fromSnapshot(
    snapshotGame(
      participants: players,
      phase: phase,
      turnUserId: turn,
      deadline: now.add(const Duration(seconds: 75)).toIso8601String(),
      round: round,
      myCards: myCardsJson(revealed: revealed),
      myActionState: state,
      tiebreak: tiebreak,
      readyCount: readyCount,
      voteProgress: votes,
      lastRoundResult: last,
    ),
    serverTime: now,
    receivedAt: now,
  );

  late List<GameAction> actions;
  setUp(() => actions = []);

  Future<void> pump(
    WidgetTester tester,
    GameView g, {
    String me = 'u1',
    Future<void> Function(GameAction)? onAction,
  }) async {
    await pumpWithProviders(
      tester,
      Scaffold(
        body: GameBody(
          game: g,
          myUserId: me,
          clock: () => now,
          onAction: onAction ?? (a) async => actions.add(a),
        ),
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
    expect(find.text('Oylamayla elendi'), findsOneWidget);
    expect(find.text('Kimse oy vermedi; herkes eşit sayıldı.'), findsOneWidget);
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

  group('eylemler', () {
    Future<void> tapAndSettle(WidgetTester tester, Finder f) async {
      await tester.tap(f);
      await tester.pumpAndSettle();
    }

    testWidgets('Kart açma: 1. turda yalnızca meslek; onaylı', (tester) async {
      await pump(tester, game(turn: 'u1', revealed: {}));

      expect(
        find.text('İlk turda yalnızca meslek kartını açabilirsin.'),
        findsOneWidget,
      );
      ActionChip chip(String c) =>
          tester.widget<ActionChip>(find.byKey(ValueKey('reveal-$c')));
      expect(chip('profession').onPressed, isNotNull);
      expect(chip('health').onPressed, isNull);

      await tapAndSettle(
        tester,
        find.byKey(const ValueKey('reveal-profession')),
      );
      expect(
        find.textContaining('Açılan kart geri kapatılamaz'),
        findsOneWidget,
      );
      await tapAndSettle(tester, find.text('Vazgeç'));
      expect(actions, isEmpty);

      await tapAndSettle(
        tester,
        find.byKey(const ValueKey('reveal-profession')),
      );
      await tapAndSettle(tester, find.text('Aç'));
      expect(actions.single, isA<RevealCard>());
      expect((actions.single as RevealCard).category, CardCategory.profession);
    });

    testWidgets('2. turda açılmamış her kart seçilebilir', (tester) async {
      await pump(tester, game(turn: 'u1', round: 2));

      expect(find.byKey(const ValueKey('reveal-profession')), findsNothing);
      await tapAndSettle(tester, find.byKey(const ValueKey('reveal-secret')));
      await tapAndSettle(tester, find.text('Aç'));
      expect((actions.single as RevealCard).category, CardCategory.secret);
    });

    testWidgets('Sıra başkasındaysa kart düğmesi yok', (tester) async {
      await pump(tester, game());
      expect(find.byType(ActionChip), findsOneWidget); // yalnızca senaryo
    });

    testWidgets('Tartışma: Hazırım; sonra bekleme metni', (tester) async {
      await pump(tester, game(phase: 'DISCUSSION'));
      await tapAndSettle(tester, find.text('Hazırım'));
      expect(actions.single, isA<DiscussionReady>());

      await pump(
        tester,
        game(
          phase: 'DISCUSSION',
          state: {
            'voted': false,
            'discussion_ready': true,
            'tiebreak_voted': false,
          },
        ),
      );
      expect(find.text('Hazırım'), findsNothing);
      expect(find.text('Hazırsın. Diğerleri bekleniyor.'), findsOneWidget);
    });

    testWidgets('Oylama: hayattaki diğerine dokun → onay → oy', (tester) async {
      await pump(tester, game(phase: 'VOTING'));
      expect(find.text('Elemek istediğin oyuncuya dokun.'), findsOneWidget);

      // Kendime ve elenmişe dokunmak bir şey yapmaz.
      await tapAndSettle(tester, find.byKey(const ValueKey('player-u1')));
      await tapAndSettle(tester, find.byKey(const ValueKey('player-u3')));
      expect(find.byType(AlertDialog), findsNothing);

      await tapAndSettle(tester, find.byKey(const ValueKey('player-u2')));
      expect(find.text('Mehmet için oy ver?'), findsOneWidget);
      await tapAndSettle(tester, find.text('Oy ver'));
      expect((actions.single as Vote).targetUserId, 'u2');
    });

    testWidgets('Oy verdiysem dokunmak bir şey yapmaz', (tester) async {
      await pump(
        tester,
        game(
          phase: 'VOTING',
          state: {
            'voted': true,
            'discussion_ready': false,
            'tiebreak_voted': false,
          },
        ),
      );
      expect(find.text('Oyunu verdin.'), findsOneWidget);
      await tapAndSettle(tester, find.byKey(const ValueKey('player-u2')));
      expect(find.byType(AlertDialog), findsNothing);
    });

    testWidgets('Eşitlik: elenmiş katılımcı adaylardan seçer', (tester) async {
      final tb = {
        'candidates': ['u1', 'u2'],
        'progress': {'cast': 0, 'total': 1},
      };
      await pump(
        tester,
        game(phase: 'TIE_BREAK', tiebreak: tb),
        me: 'u3',
      );
      expect(
        find.text('Eşitlik! Kararı izleyiciler veriyor; sen de oy ver.'),
        findsOneWidget,
      );
      expect(find.text('0/1 oy verildi'), findsOneWidget);

      await tapAndSettle(tester, find.byKey(const ValueKey('tiebreak-u1')));
      await tapAndSettle(tester, find.text('Oy ver'));
      expect((actions.single as TiebreakVote).targetUserId, 'u1');

      // Hayattaki aday ise düğme görmez.
      await pump(tester, game(phase: 'TIE_BREAK', tiebreak: tb));
      expect(find.byKey(const ValueKey('tiebreak-u1')), findsNothing);
      expect(find.text('Eşitlik! İzleyiciler karar veriyor…'), findsOneWidget);
    });

    testWidgets('İstek sürerken ikinci eylem gönderilmez', (tester) async {
      final gate = Completer<void>();
      var calls = 0;
      await pump(
        tester,
        game(phase: 'DISCUSSION'),
        onAction: (a) {
          calls++;
          return gate.future;
        },
      );

      await tester.tap(find.text('Hazırım'));
      await tester.pump();
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      await tester.tap(find.byType(FilledButton).first);
      await tester.pump();
      expect(calls, 1);

      gate.complete();
      await tester.pumpAndSettle();
    });
  });

  group('eleme', () {
    Map<String, Object?> elim(
      String user,
      String method, {
      List<Object?>? tally,
      List<Object?>? lottery,
    }) => {
      'round': 1,
      'user_id': user,
      'method': method,
      'tally': tally ?? <Object?>[],
      'lottery_candidates': lottery ?? <Object?>[],
      'cards': <Object?>[],
    };

    testWidgets('Oy sayıları büyükten küçüğe, elenenin kartları', (
      tester,
    ) async {
      await pump(
        tester,
        game(
          phase: 'ELIMINATION',
          last: elim(
            'u3',
            'vote',
            tally: [
              {'user_id': 'u2', 'votes': 1},
              {'user_id': 'u3', 'votes': 3},
            ],
          ),
        ),
      );

      expect(find.byKey(const ValueKey('elimination-panel')), findsOneWidget);
      final zeynep = tester.getTopLeft(find.text('3 oy'));
      final mehmet = tester.getTopLeft(find.text('1 oy'));
      expect(zeynep.dy, lessThan(mehmet.dy));
      expect(find.text('Kartları'), findsOneWidget);
      expect(find.text('Sır'), findsOneWidget);
    });

    testWidgets('Kura: adaylar yazılır', (tester) async {
      await pump(
        tester,
        game(
          phase: 'ELIMINATION',
          last: elim('u3', 'lottery', lottery: ['u2', 'u3']),
        ),
      );
      expect(find.text('Eşitlik kurayla bozuldu'), findsOneWidget);
      expect(find.text('Kura adayları: Mehmet, Zeynep'), findsOneWidget);
    });

    testWidgets('İzleyici kararı: oy listesi yok', (tester) async {
      await pump(
        tester,
        game(
          phase: 'ELIMINATION',
          last: elim(
            'u3',
            'spectators',
            tally: [
              {'user_id': 'u3', 'votes': 1},
            ],
          ),
        ),
      );
      expect(find.text('İzleyicilerin kararıyla elendi'), findsOneWidget);
      expect(find.text('1 oy'), findsNothing);
    });

    testWidgets('Elendiğim an "Kapı kapandı" 2 sn görünür', (tester) async {
      final alive = game(phase: 'VOTING');
      await pump(tester, alive);
      expect(find.byKey(const ValueKey('door-closed')), findsNothing);

      final out = alive.apply(
        'game.player_eliminated',
        elim('u1', 'vote'),
        receivedAt: now,
        myUserId: 'u1',
      );
      await pump(tester, out);
      expect(find.text('Kapı kapandı.'), findsOneWidget);
      expect(find.text('Artık izleyicisin.'), findsOneWidget);

      await tester.pump(const Duration(seconds: 2));
      expect(find.byKey(const ValueKey('door-closed')), findsNothing);
    });

    testWidgets('Başkası elenince veya ayrılınca kapı ekranı yok', (
      tester,
    ) async {
      final alive = game(phase: 'VOTING');
      await pump(tester, alive);
      await pump(
        tester,
        alive.apply(
          'game.player_eliminated',
          elim('u2', 'vote'),
          receivedAt: now,
          myUserId: 'u1',
        ),
      );
      expect(find.byKey(const ValueKey('door-closed')), findsNothing);

      await pump(tester, alive);
      await pump(
        tester,
        alive.apply(
          'game.player_eliminated',
          elim('u1', 'left'),
          receivedAt: now,
          myUserId: 'u1',
        ),
      );
      expect(find.byKey(const ValueKey('door-closed')), findsNothing);
    });
  });
}
