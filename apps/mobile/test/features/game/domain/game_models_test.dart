import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/features/game/game.dart';

import '../../../helpers/fake_ws.dart';

void main() {
  final t0 = DateTime.utc(2026, 11, 4, 10);
  final players = [
    participantJson('u1', 'Ayşe', 0, revealed: {'profession': 'Doktor'}),
    participantJson('u2', 'Mehmet', 1),
    participantJson('u3', 'Zeynep', 2, status: 'eliminated'),
  ];

  GameView snap({String phase = 'REVEAL', int round = 1}) =>
      GameView.fromSnapshot(
        snapshotGame(
          participants: players,
          phase: phase,
          round: round,
          turnUserId: 'u2',
          deadline: t0.add(const Duration(seconds: 20)).toIso8601String(),
          myCards: myCardsJson(revealed: {'profession'}),
          readyCount: 1,
        ),
        serverTime: t0,
        receivedAt: t0,
      );

  GameView apply(GameView g, String type, Map<String, Object?> p) =>
      g.apply(type, p, receivedAt: t0, myUserId: 'u1');

  test('snapshot çözülür', () {
    final g = snap();
    expect(g.phase, GamePhase.reveal);
    expect(g.round, 1);
    expect(g.turnUserId, 'u2');
    expect(g.scenario!.bunker, hasLength(3));
    expect(g.participants.map((p) => p.nickname), ['Ayşe', 'Mehmet', 'Zeynep']);
    expect(g.participants[0].revealed, {CardCategory.profession: 'Doktor'});
    expect(g.participants[2].status, ParticipantStatus.eliminated);
    expect(g.aliveCount, 2);
    expect(g.myCards.map((c) => c.category), CardCategory.values);
    expect(g.myCards.first.revealed, isTrue);
    expect(g.readyProgress, (done: 1, total: 2));
  });

  test('katılımcılar koltuk sırasına dizilir', () {
    final g = GameView.fromSnapshot(
      snapshotGame(
        participants: [
          participantJson('b', 'B', 1),
          participantJson('a', 'A', 0),
        ],
      ),
      serverTime: t0,
      receivedAt: t0,
    );
    expect(g.participants.map((p) => p.userId), ['a', 'b']);
  });

  test('bilinmeyen aşama → FormatException', () {
    expect(
      () => GameView.fromSnapshot(
        snapshotGame(participants: players, phase: 'NAP'),
        serverTime: t0,
        receivedAt: t0,
      ),
      throwsFormatException,
    );
  });

  test('kalan süre sunucu saatine göre (cihaz saati kaymış)', () {
    // Cihaz saati sunucudan 5 sn geride.
    final g = GameView.fromSnapshot(
      snapshotGame(
        participants: players,
        deadline: t0.add(const Duration(seconds: 20)).toIso8601String(),
      ),
      serverTime: t0,
      receivedAt: t0.subtract(const Duration(seconds: 5)),
    );
    final deviceNow = t0.subtract(const Duration(seconds: 5));
    expect(g.remaining(deviceNow), const Duration(seconds: 20));
    expect(
      g.remaining(deviceNow.add(const Duration(seconds: 30))),
      Duration.zero,
    );
  });

  test('game.started + your_cards', () {
    var g = GameView.fromStarted(
      gameStartedPayload([('u1', 'Ayşe'), ('u2', 'Mehmet')]),
      receivedAt: t0,
    );
    expect(g.phase, GamePhase.dealing);
    expect(g.scenario!.title, 'Göktaşı');
    expect(g.myCards, isEmpty);
    expect(g.aliveCount, 2);

    g = apply(g, 'game.your_cards', {'cards': myCardsJson()});
    expect(g.myCards, hasLength(5));
    expect(g.myCards.every((c) => !c.revealed), isTrue);
  });

  test('phase_changed: sıra, süre, yeni turda bayraklar sıfırlanır', () {
    var g = GameView.fromSnapshot(
      snapshotGame(
        participants: players,
        phase: 'VOTING',
        myActionState: {
          'voted': true,
          'discussion_ready': true,
          'tiebreak_voted': false,
        },
      ),
      serverTime: t0,
      receivedAt: t0,
    );
    expect(g.voted, isTrue);

    g = apply(g, 'game.phase_changed', {
      'phase': 'REVEAL',
      'round': 2,
      'phase_deadline_at': '2026-11-04T10:00:20Z',
      'server_time': '2026-11-04T10:00:00Z',
      'turn_user_id': 'u1',
    });
    expect(g.phase, GamePhase.reveal);
    expect(g.round, 2);
    expect(g.turnUserId, 'u1');
    expect(g.voted, isFalse);
    expect(g.discussionReady, isFalse);
    expect(g.remaining(t0), const Duration(seconds: 20));

    g = apply(g, 'game.phase_changed', {
      'phase': 'DISCUSSION',
      'round': 2,
      'phase_deadline_at': null,
      'turn_user_id': null,
    });
    expect(g.turnUserId, isNull);
    expect(g.readyProgress, (done: 0, total: 2));
  });

  test('card_revealed: oyuncuya ve kendi kartıma işlenir', () {
    var g = snap();
    g = apply(g, 'game.card_revealed', {
      'user_id': 'u1',
      'category': 'health',
      'text': 'Astım',
      'auto': false,
    });
    expect(g.participantOf('u1')!.revealed[CardCategory.health], 'Astım');
    expect(
      g.myCards.firstWhere((c) => c.category == CardCategory.health).revealed,
      isTrue,
    );

    g = apply(g, 'game.card_revealed', {
      'user_id': 'u2',
      'category': 'item',
      'text': 'Termos',
    });
    expect(g.participantOf('u2')!.revealed, {CardCategory.item: 'Termos'});
    expect(
      g.myCards.firstWhere((c) => c.category == CardCategory.item).revealed,
      isFalse,
    );
  });

  test('bozuk card_revealed → FormatException', () {
    expect(
      () => apply(snap(), 'game.card_revealed', {'user_id': 'u1'}),
      throwsFormatException,
    );
  });

  test('ilerleme olayları', () {
    var g = snap(phase: 'DISCUSSION');
    g = apply(g, 'game.discussion_ready', {'count': 2, 'total': 2});
    expect(g.readyProgress, (done: 2, total: 2));
    g = apply(g, 'game.vote_progress', {'cast': 1, 'total': 2});
    expect(g.voteProgress, (done: 1, total: 2));
    g = apply(g, 'game.tiebreak_started', {
      'candidates': ['u1', 'u2'],
      'phase_deadline_at': '2026-11-04T10:01:00Z',
    });
    expect(g.tiebreakCandidates, ['u1', 'u2']);
    expect(g.remaining(t0), const Duration(minutes: 1));
    g = apply(g, 'game.tiebreak_progress', {'cast': 1, 'total': 1});
    expect(g.tiebreakProgress, (done: 1, total: 1));
  });

  test('eleme: durum, tüm kartlar ve sonuç', () {
    var g = snap();
    g = apply(g, 'game.player_eliminated', {
      'round': 1,
      'user_id': 'u2',
      'method': 'vote',
      'tally': [
        {'user_id': 'u2', 'votes': 2},
      ],
      'lottery_candidates': <String>[],
      'cards': [
        {'category': 'profession', 'text': 'Aşçı'},
        {'category': 'secret', 'text': 'Sır'},
      ],
    });
    final p = g.participantOf('u2')!;
    expect(p.status, ParticipantStatus.eliminated);
    expect(p.revealed, hasLength(2));
    expect(g.lastElimination!.tally.single, (userId: 'u2', votes: 2));
    expect(g.aliveCount, 1);
  });

  test('ben elenirsem tüm kartlarım açık sayılır', () {
    final g = apply(snap(), 'game.player_eliminated', {
      'round': 1,
      'user_id': 'u1',
      'method': 'lottery',
      'tally': <Object?>[],
      'lottery_candidates': ['u1', 'u2'],
      'cards': <Object?>[],
    });
    expect(g.myCards.every((c) => c.revealed), isTrue);
    expect(g.lastElimination!.lotteryCandidates, ['u1', 'u2']);
  });

  test('ayrılma eleme sonucunu değiştirmez; durum left', () {
    final g = apply(snap(), 'game.player_eliminated', {
      'round': 1,
      'user_id': 'u2',
      'method': 'left',
      'tally': null,
      'lottery_candidates': null,
      'cards': <Object?>[],
    });
    expect(g.participantOf('u2')!.status, ParticipantStatus.left);
    expect(g.lastElimination, isNull);
  });

  test('bağlantı olayları ve bitiş', () {
    var g = snap();
    g = apply(g, 'game.player_connection', {
      'user_id': 'u2',
      'state': 'disconnected',
    });
    expect(g.participantOf('u2')!.connected, isFalse);
    g = apply(g, 'game.player_connection', {
      'user_id': 'u2',
      'state': 'reconnected',
    });
    expect(g.participantOf('u2')!.connected, isTrue);
    g = apply(g, 'game.player_connection', {'user_id': 'u2', 'state': 'left'});
    expect(g.participantOf('u2')!.status, ParticipantStatus.left);

    g = apply(g, 'game.finished', {});
    expect(g.phase, GamePhase.finished);
    expect(g.remaining(t0), isNull);
  });

  test('bilinmeyen olay değişiklik yapmaz', () {
    final g = snap();
    expect(identical(apply(g, 'game.yeni_olay', {}), g), isTrue);
  });
}
