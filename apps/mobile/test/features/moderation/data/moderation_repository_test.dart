import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/features/moderation/moderation.dart';

import '../../../helpers/fake_http_adapter.dart';

void main() {
  late List<RequestOptions> sent;
  late Object? reply;

  ModerationRepository repo() {
    sent = [];
    return ModerationRepository(
      ApiClient(
        baseUrl: Uri.parse('http://localhost'),
        appVersion: '0.1.0',
        readToken: () async => 't',
        adapter: FakeHttpAdapter((r) async {
          sent.add(r);
          return reply == null
              ? ResponseBodyNoContent.value
              : jsonResponse(reply, r.method == 'POST' ? 201 : 200);
        }),
        enableLogging: false,
      ),
    );
  }

  test('engellenenler listesi', () async {
    reply = [
      {
        'user_id': 'u2',
        'nickname': 'Mehmet',
        'created_at': '2026-11-04T10:00:00Z',
      },
    ];
    final list = await repo().blocks();
    expect(list.single.userId, 'u2');
    expect(list.single.nickname, 'Mehmet');
    expect(sent.single.path, '/v1/blocks');
  });

  test('bozuk liste invalidResponse', () async {
    reply = {'x': 1};
    await expectLater(
      repo().blocks(),
      throwsA(
        isA<ApiException>().having(
          (e) => e.kind,
          'kind',
          ApiErrorKind.invalidResponse,
        ),
      ),
    );
    reply = [
      {'nickname': 'kimliksiz'},
    ];
    await expectLater(repo().blocks(), throwsA(isA<ApiException>()));
  });

  test('engelle, engeli kaldır, raporla', () async {
    reply = null;
    final r = repo();
    await r.block('u2');
    await r.unblock('u2');
    await r.report(
      reportedUserId: 'u2',
      roomId: 'r1',
      reason: ReportReason.harassment,
      note: '  hakaret etti  ',
    );
    await r.report(
      reportedUserId: 'u3',
      roomId: 'r1',
      reason: ReportReason.other,
      note: '   ',
    );

    expect(sent.map((s) => '${s.method} ${s.path}'), [
      'POST /v1/blocks',
      'DELETE /v1/blocks/u2',
      'POST /v1/reports',
      'POST /v1/reports',
    ]);
    expect(sent[0].data, {'user_id': 'u2'});
    expect(sent[2].data, {
      'reported_user_id': 'u2',
      'room_id': 'r1',
      'reason': 'harassment',
      'note': 'hakaret etti',
    });
    expect((sent[3].data as Map).containsKey('note'), isFalse);
  });
}
