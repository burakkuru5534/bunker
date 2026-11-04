import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/features/chat/chat.dart';

import '../../../helpers/fake_http_adapter.dart';

void main() {
  test('gövde ve yanıt', () async {
    final adapter = FakeHttpAdapter(
      (r) async =>
          jsonResponse({'id': 42, 'created_at': '2026-11-04T10:00:00Z'}, 201),
    );
    final repo = ChatRepository(
      ApiClient(
        baseUrl: Uri.parse('http://localhost'),
        appVersion: '0.1.0',
        readToken: () async => 't',
        adapter: adapter,
        enableLogging: false,
      ),
    );

    final sent = await repo.send('r1', clientMsgId: 'c1', text: 'selam');

    expect(sent.id, 42);
    expect(adapter.requests.single.path, '/v1/rooms/r1/messages');
    expect(adapter.requests.single.data, {
      'client_msg_id': 'c1',
      'text': 'selam',
    });
  });

  test('bozuk yanıt invalidResponse', () async {
    final repo = ChatRepository(
      ApiClient(
        baseUrl: Uri.parse('http://localhost'),
        appVersion: '0.1.0',
        readToken: () async => 't',
        adapter: FakeHttpAdapter((r) async => jsonResponse({'id': 'x'}, 201)),
        enableLogging: false,
      ),
    );
    await expectLater(
      repo.send('r1', clientMsgId: 'c', text: 't'),
      throwsA(
        isA<ApiException>().having(
          (e) => e.kind,
          'kind',
          ApiErrorKind.invalidResponse,
        ),
      ),
    );
  });
}
