import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/features/game/game.dart';
import 'package:siginak/features/game/presentation/intro/intro_timeline.dart';

import '../../../helpers/fake_intro_audio.dart';
import '../../../helpers/fake_ws.dart';
import '../../../helpers/pump_app.dart';
import '../../../helpers/small_screen.dart';

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
    Duration left = const Duration(seconds: 75),
    String scenarioId = 'meteor',
  }) => GameView.fromSnapshot(
    {
      ...snapshotGame(
        participants: players,
        phase: phase,
        turnUserId: turn,
        deadline: now.add(left).toIso8601String(),
        round: round,
        myCards: myCardsJson(revealed: revealed),
        myActionState: state,
        tiebreak: tiebreak,
        readyCount: readyCount,
        voteProgress: votes,
        lastRoundResult: last,
      ),
      'scenario': {...testScenario, 'id': scenarioId},
    },
    serverTime: now,
    receivedAt: now,
  );

  late List<GameAction> actions;
  late List<FakeIntroAudio> audios;
  setUp(() {
    actions = [];
    audios = [];
  });

  Future<void> pump(
    WidgetTester tester,
    GameView g, {
    String me = 'u1',
    Future<void> Function(GameAction)? onAction,
  }) async {
    await pumpWithProviders(
      tester,
      introAudio: () => FakeIntroAudio(audios),
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
    await tester.tap(find.byKey(const ValueKey('intro-skip')));
    await tester.pump();
    expect(find.text('Sığınakta neler var?'), findsOneWidget);
    expect(find.text('Teknik kütüphane'), findsOneWidget);
    expect(find.text('Sığınağa yalnızca 2 kişi girebilir.'), findsOneWidget);

    await pump(tester, game());
    expect(find.text('Sığınakta neler var?'), findsNothing);
    await tester.tap(find.text('Göktaşı'));
    await tester.pumpAndSettle();
    expect(find.text('Sığınakta neler var?'), findsOneWidget);
  });

  group('Senaryo tanıtımı animasyonu', () {
    final cinematic = find.byKey(const ValueKey('scenario-cinematic'));
    final subtitle = find.byKey(const ValueKey('intro-subtitle'));

    String? subtitleText(WidgetTester tester) =>
        subtitle.evaluate().isEmpty ? null : tester.widget<Text>(subtitle).data;

    testWidgets('baştan oynar: başlık, altyazı, ses; Geç ile kapanır', (
      tester,
    ) async {
      await pump(
        tester,
        game(phase: 'SCENARIO_INTRO', turn: null, left: kIntroLength),
      );
      expect(cinematic, findsOneWidget);
      expect(audios.single.scenarioId, 'meteor');
      expect(audios.single.startedAt, 0);
      expect(audios.single.muted, [false]);

      await tester.pump(const Duration(milliseconds: 1500));
      expect(find.text('GÖKTAŞI'), findsOneWidget);
      expect(subtitleText(tester), isNull);

      await tester.pump(const Duration(seconds: 1));
      expect(subtitleText(tester), 'O gece gökyüzü bir anda aydınlandı.');

      await tester.tap(find.byKey(const ValueKey('intro-skip')));
      await tester.pump();
      expect(cinematic, findsNothing);
      expect(audios.single.disposed, isTrue);
      expect(find.text('Sığınakta neler var?'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });

    testWidgets('sonradan katılan ortadan başlar', (tester) async {
      await pump(
        tester,
        game(
          phase: 'SCENARIO_INTRO',
          turn: null,
          left: const Duration(seconds: 8),
        ),
      );
      expect(audios.single.startedAt, 22);
      await tester.pump(const Duration(milliseconds: 100));
      expect(subtitleText(tester), contains('Kim içeri girecek?'));
    });

    testWidgets('süre bitmek üzereyse gösterilmez', (tester) async {
      await pump(
        tester,
        game(
          phase: 'SCENARIO_INTRO',
          turn: null,
          left: const Duration(seconds: 1),
        ),
      );
      expect(cinematic, findsNothing);
      expect(audios, isEmpty);
    });

    testWidgets('bilinmeyen senaryoda metin tanıtımı görünür', (tester) async {
      await pump(
        tester,
        game(phase: 'SCENARIO_INTRO', turn: null, scenarioId: 'yeni'),
      );
      expect(cinematic, findsNothing);
      expect(find.text('Sığınakta neler var?'), findsOneWidget);
      expect(audios, isEmpty);
    });

    testWidgets('aşama bitince kapanır; süre azaldı diye erken kapanmaz', (
      tester,
    ) async {
      await pump(
        tester,
        game(phase: 'SCENARIO_INTRO', turn: null, left: kIntroLength),
      );
      // Aynı oyun, aynı aşama; kalan süre 1 sn → animasyon sürer.
      await pump(
        tester,
        game(
          phase: 'SCENARIO_INTRO',
          turn: null,
          left: const Duration(seconds: 1),
        ),
      );
      expect(cinematic, findsOneWidget);
      expect(audios, hasLength(1));

      await pump(tester, game());
      expect(cinematic, findsNothing);
      expect(audios.single.disposed, isTrue);
    });

    testWidgets('ses düğmesi sessize alır ve geri açar', (tester) async {
      await pump(tester, game(phase: 'SCENARIO_INTRO', turn: null));
      final mute = find.byKey(const ValueKey('intro-mute'));
      expect(find.byTooltip('Sesi kapat'), findsOneWidget);

      await tester.tap(mute);
      await tester.pump();
      expect(audios.single.muted, [false, true]);
      expect(find.byTooltip('Sesi aç'), findsOneWidget);

      await tester.tap(mute);
      await tester.pump();
      expect(audios.single.muted, [false, true, false]);
    });

    testWidgets('320 dp ve %200 yazıda her anda taşma yok', (tester) async {
      useSmallScreen(tester);
      await pump(
        tester,
        game(phase: 'SCENARIO_INTRO', turn: null, left: kIntroLength),
      );
      for (var t = 0; t < 30; t++) {
        await tester.pump(const Duration(seconds: 1));
        expect(tester.takeException(), isNull, reason: 't=$t');
      }
    });
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

  testWidgets('320 dp ve %200 yazı: tüm aşamalarda taşma yok', (tester) async {
    useSmallScreen(tester);
    final elimination = {
      'round': 1,
      'user_id': 'u3',
      'method': 'vote',
      'tally': [
        {'user_id': 'u2', 'votes': 1},
        {'user_id': 'u3', 'votes': 3},
      ],
      'lottery_candidates': <Object?>[],
      'cards': <Object?>[],
    };
    final cases = <String, (GameView, String)>{
      'tanıtım': (game(phase: 'SCENARIO_INTRO'), 'u1'),
      'kart açma (sıra bende)': (game(turn: 'u1', revealed: {}), 'u1'),
      'tartışma': (game(phase: 'DISCUSSION'), 'u1'),
      'oylama': (game(phase: 'VOTING'), 'u1'),
      'eşitlik (izleyici)': (
        game(
          phase: 'TIE_BREAK',
          tiebreak: {
            'candidates': ['u1', 'u2'],
            'progress': {'cast': 0, 'total': 1},
          },
        ),
        'u3',
      ),
      'eleme': (game(phase: 'ELIMINATION', last: elimination), 'u1'),
    };
    for (final MapEntry(key: name, value: (g, me)) in cases.entries) {
      await pump(tester, g, me: me);
      expect(tester.takeException(), isNull, reason: name);
      // Üst çubuktaki tur, aşama ve süre kırpılmadan kendi alanında.
      final box = find.byKey(const ValueKey('phase-status'));
      final status = tester.getRect(box);
      final texts = find.descendant(of: box, matching: find.byType(Text));
      expect(texts, findsAtLeastNWidgets(2), reason: name);
      for (final text in texts.evaluate()) {
        final r = tester.getRect(find.byWidget(text.widget));
        expect(
          status.left <= r.left + 0.5 && r.right <= status.right + 0.5,
          isTrue,
          reason: '$name: $r ⊄ $status',
        );
      }
    }
    // Kartlarım sayfası ve senaryo penceresi.
    await pump(tester, game());
    await tester.tap(find.text('Kartlarım'));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull, reason: 'Kartlarım');
    tester.state<NavigatorState>(find.byType(Navigator)).pop();
    await tester.pumpAndSettle();
    await tester.tap(find.text('Göktaşı'));
    await tester.pumpAndSettle();
    expect(find.text('Sığınakta neler var?'), findsOneWidget);
    expect(tester.takeException(), isNull, reason: 'senaryo penceresi');
  });

  testWidgets('Elenenin 5 kartı açıkken kutu taşmaz (normal ve %200 yazı)', (
    tester,
  ) async {
    final all = {
      'profession': 'Uzun adlı bir meslek kartı metni',
      'health': 'Şiddetli fıstık alerjisi',
      'hobby': 'Dağcılık',
      'item': 'İsviçre çakısı',
      'secret': 'Sığınağın kapısını açık bırakan oydu',
    };
    final g = GameView.fromSnapshot(
      snapshotGame(
        participants: [
          participantJson('u1', 'Ayşe', 0),
          participantJson(
            'u2',
            'Mehmet',
            1,
            status: 'eliminated',
            revealed: all,
          ),
          participantJson('u3', 'Zeynep', 2),
        ],
        phase: 'DISCUSSION',
        deadline: now.add(const Duration(seconds: 75)).toIso8601String(),
        myCards: myCardsJson(),
      ),
      serverTime: now,
      receivedAt: now,
    );
    for (final (size, scale) in [
      (const Size(360, 640), 1.0),
      (smallScreen, 2.0),
    ]) {
      useSmallScreen(tester, size: size, textScale: scale);
      await pump(tester, g);
      expect(tester.takeException(), isNull, reason: '$size × $scale');
      expect(find.text('Sığınağın kapısını açık bırakan oydu'), findsOneWidget);
    }
  });

  testWidgets('Dokunma hedefleri ve etiketler (360 dp)', (tester) async {
    useSmallScreen(tester, size: const Size(360, 640), textScale: 1);
    await pump(tester, game(turn: 'u1', revealed: {}));
    await expectAccessibleTapTargets(tester);
    await pump(tester, game(phase: 'VOTING'));
    await expectAccessibleTapTargets(tester);
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

  testWidgets('Oyuncu menüsü: başkasına dokununca açılır, kendime değil', (
    tester,
  ) async {
    final menus = <String>[];
    await pumpWithProviders(
      tester,
      Scaffold(
        body: GameBody(
          game: game(phase: 'DISCUSSION'),
          myUserId: 'u1',
          clock: () => now,
          onAction: (a) async {},
          onPlayerMenu: (p) => menus.add(p.userId),
        ),
      ),
    );
    await tester.pump();

    await tester.tap(find.byKey(const ValueKey('player-u1')));
    await tester.tap(find.byKey(const ValueKey('player-u2')));
    await tester.pump();

    expect(menus, ['u2']);
  });
}
