import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/features/game/game.dart';

import '../../../helpers/fake_backend.dart';
import '../../../helpers/fake_http_adapter.dart';
import '../../../helpers/fake_ws.dart';

void main() {
  late List<RequestOptions> sent;
  late Future<ResponseBody> Function(RequestOptions) handler;
  late GameRepository repo;
  var n = 0;

  setUp(() {
    sent = [];
    n = 0;
    handler = (r) async =>
        jsonResponse({'state_version': 5, 'duplicate': false}, 200);
    final api = ApiClient(
      baseUrl: Uri.parse('http://localhost'),
      appVersion: '0.1.0',
      readToken: () async => 't',
      adapter: FakeHttpAdapter((r) {
        sent.add(r);
        return handler(r);
      }),
      enableLogging: false,
    );
    repo = GameRepository(api, newId: () => 'id-${++n}');
  });

  Map<String, Object?> body(int i) => sent[i].data as Map<String, Object?>;

  test('eylem gövdesi', () async {
    await repo.act('g1', 1, const RevealCard(CardCategory.profession));
    await repo.act('g1', 1, const DiscussionReady());
    await repo.act('g1', 1, const Vote('u2'));
    await repo.act('g1', 1, const TiebreakVote('u3'));

    expect(sent.map((r) => r.path).toSet(), {'/v1/games/g1/actions'});
    expect(body(0), {
      'action_id': 'id-1',
      'type': 'reveal_card',
      'payload': {'category': 'profession'},
    });
    expect(body(1)['type'], 'discussion_ready');
    expect(body(1)['payload'], <String, Object?>{});
    expect(body(2)['payload'], {'target_user_id': 'u2'});
    expect(body(3)['type'], 'tiebreak_vote');
  });

  test('ağ hatasından sonra aynı eylem aynı kimlikle gider', () async {
    handler = networkFailure;
    await expectLater(
      repo.act('g1', 2, const Vote('u2')),
      throwsA(isA<ApiException>()),
    );
    handler = (r) async =>
        jsonResponse({'state_version': 9, 'duplicate': true}, 200);
    await repo.act('g1', 2, const Vote('u2'));

    expect(body(0)['action_id'], body(1)['action_id']);

    // Başarıdan sonra (ör. yeni turda) yeni kimlik.
    await repo.act('g1', 3, const Vote('u2'));
    expect(body(2)['action_id'], isNot(body(1)['action_id']));
  });

  test('başarılı eylemin kimliği tekrar kullanılmaz', () async {
    await repo.act('g1', 1, const DiscussionReady());
    await repo.act('g1', 1, const DiscussionReady());

    expect(body(0)['action_id'], 'id-1');
    expect(body(1)['action_id'], 'id-2');
  });

  test('sonuç okunur; bozuk sonuç invalidResponse', () async {
    handler = (r) async => jsonResponse(resultJson(), 200);
    final r = await repo.fetchResult('g1');
    expect(sent.last.path, '/v1/games/g1/result');
    expect(r.winners, hasLength(2));

    // Kimliksiz oyuncu → bozuk yanıt.
    handler = (r) async => jsonResponse({
      'players': [<String, Object?>{}],
    }, 200);
    await expectLater(
      repo.fetchResult('g1'),
      throwsA(
        isA<ApiException>().having(
          (e) => e.kind,
          'kind',
          ApiErrorKind.invalidResponse,
        ),
      ),
    );
  });

  test('farklı eylem veya tur farklı kimlik alır', () async {
    handler = networkFailure;
    for (final a in [const Vote('u2'), const Vote('u3')]) {
      await expectLater(repo.act('g1', 1, a), throwsA(isA<ApiException>()));
    }
    await expectLater(
      repo.act('g1', 2, const Vote('u2')),
      throwsA(isA<ApiException>()),
    );

    expect({for (var i = 0; i < 3; i++) body(i)['action_id']}, hasLength(3));
  });
}
