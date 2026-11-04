import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/features/game/game.dart';

import '../../../helpers/fake_ws.dart';

void main() {
  test('sonuç çözülür; kartlar kategori sırasında', () {
    final r = GameResult.fromJson(resultJson());

    expect(r.gameId, 'g1');
    expect(r.scenarioTitle, 'Göktaşı');
    expect(r.winners.map((p) => p.nickname), ['Ayşe', 'Mehmet']);
    expect(r.losers.single.eliminatedRound, 1);
    expect(r.losers.single.status, ParticipantStatus.eliminated);
    expect(r.players.first.cards.map((c) => c.category), [
      CardCategory.profession,
      CardCategory.secret,
    ]);
    expect(r.rounds.single.tally.first, (userId: 'u3', votes: 2));
    expect(r.nicknameOf('u3'), 'Zeynep');
    expect(r.nicknameOf(null), '');
  });

  test('bozuk sonuç → FormatException', () {
    expect(() => GameResult.fromJson('x'), throwsFormatException);
    expect(
      () => GameResult.fromJson({
        ...resultJson(),
        'players': [
          {'nickname': 'kimliksiz'},
        ],
      }),
      throwsFormatException,
    );
  });
}
