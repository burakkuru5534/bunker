import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/features/room/room.dart';

import '../../../helpers/fake_backend.dart';

void main() {
  test('RoomView çözülür', () {
    final room = RoomView.fromJson(
      roomJson(
        status: 'in_game',
        capacity: 1,
        version: 7,
        members: [
          {
            'user_id': 'u1',
            'nickname': 'Ayşe',
            'role': 'player',
            'is_ready': true,
            'connected': true,
          },
          {
            'user_id': 'u2',
            'nickname': 'Mehmet',
            'role': 'spectator',
            'is_ready': false,
            'connected': false,
          },
        ],
      ),
    );

    expect(room.id, FakeBackend.createdRoomId);
    expect(room.inviteCode, 'K7M2QX');
    expect(room.status, RoomStatus.inGame);
    expect(room.capacity, 1);
    expect(room.version, 7);
    expect(room.currentGameId, isNull);
    expect(room.scenarioId, isNull); // alan yok → rastgele
    expect(
      RoomView.fromJson({...roomJson(), 'scenario_id': 'meteor'}).scenarioId,
      'meteor',
    );
    expect(room.members, hasLength(2));
    expect(room.members[0].isReady, isTrue);
    expect(room.members[1].role, MemberRole.spectator);
    expect(() => room.members.add(room.members[0]), throwsUnsupportedError);
  });

  test('bozuk RoomView invalidResponse', () {
    final broken = <Map<String, Object?>>[
      {...roomJson(), 'status': 'closed'},
      {...roomJson(), 'capacity': '2'},
      {...roomJson(), 'members': null},
      {...roomJson(), 'current_game_id': 5},
      {...roomJson(), 'scenario_id': 7},
      {
        ...roomJson(),
        'members': [
          {'user_id': 'u1', 'nickname': 'A', 'role': 'king'},
        ],
      },
    ];
    for (final json in broken) {
      expect(
        () => RoomView.fromJson(json),
        throwsA(
          isA<ApiException>().having(
            (e) => e.kind,
            'kind',
            ApiErrorKind.invalidResponse,
          ),
        ),
        reason: '$json',
      );
    }
  });

  test('ScenarioOption listesi çözülür; bozuksa invalidResponse', () {
    final list = ScenarioOption.listFromJson({
      'scenarios': [
        {'id': 'meteor', 'title': 'Göktaşı'},
        {'id': 'red_cough', 'title': 'Kızıl Öksürük'},
      ],
    });
    expect(list, const [
      ScenarioOption(id: 'meteor', title: 'Göktaşı'),
      ScenarioOption(id: 'red_cough', title: 'Kızıl Öksürük'),
    ]);
    for (final bad in <Object?>[
      null,
      {'scenarios': null},
      {
        'scenarios': [
          {'id': 'x'},
        ],
      },
    ]) {
      expect(
        () => ScenarioOption.listFromJson(bad),
        throwsA(
          isA<ApiException>().having(
            (e) => e.kind,
            'kind',
            ApiErrorKind.invalidResponse,
          ),
        ),
      );
    }
  });
}
