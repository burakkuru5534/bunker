import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/network/api_exception.dart';

import '../../helpers/fake_http_adapter.dart';

void main() {
  late FakeHttpAdapter adapter;
  String? token;
  var unauthorizedCalls = 0;
  var upgradeCalls = 0;

  ApiClient client(Future<ResponseBody> Function(RequestOptions) handler) {
    adapter = FakeHttpAdapter(handler);
    return ApiClient(
      baseUrl: Uri.parse('https://api.example.com'),
      appVersion: '9.9.9',
      readToken: () async => token,
      onUnauthorized: () => unauthorizedCalls++,
      onUpgradeRequired: () => upgradeCalls++,
      adapter: adapter,
      enableLogging: false,
    );
  }

  setUp(() {
    token = null;
    unauthorizedCalls = 0;
    upgradeCalls = 0;
  });

  Future<ApiException> expectApiError(Future<Object?> call) async {
    try {
      await call;
    } on ApiException catch (e) {
      return e;
    }
    fail('ApiException bekleniyordu');
  }

  group('istek', () {
    test('taban adres, sürüm başlığı ve JSON gövdesi gönderilir', () async {
      final api = client((_) async => jsonResponse({'ok': true}, 200));

      final result = await api.post('/v1/rooms', body: {'capacity': 2});

      final req = adapter.requests.single;
      expect(req.uri.toString(), 'https://api.example.com/v1/rooms');
      expect(req.method, 'POST');
      expect(req.headers['X-App-Version'], '9.9.9');
      expect(req.headers['Accept'], 'application/json');
      expect(req.contentType, startsWith('application/json'));
      expect(req.data, {'capacity': 2});
      expect(result, {'ok': true});
    });

    test('token varsa Authorization eklenir, yoksa eklenmez', () async {
      final api = client((_) async => jsonResponse({}, 200));

      await api.get('/v1/me');
      expect(
        adapter.requests.last.headers.containsKey('Authorization'),
        isFalse,
      );

      token = 'abc';
      await api.get('/v1/me');
      expect(adapter.requests.last.headers['Authorization'], 'Bearer abc');
    });

    test('sorgu parametreleri ve PATCH/DELETE yöntemleri', () async {
      final api = client((_) async => jsonResponse({}, 200));

      await api.get('/v1/blocks', query: {'limit': 10});
      await api.patch('/v1/me', body: {'nickname': 'Ayşe'});
      await api.delete('/v1/blocks/x');

      expect(adapter.requests[0].uri.query, 'limit=10');
      expect(adapter.requests[1].method, 'PATCH');
      expect(adapter.requests[2].method, 'DELETE');
    });

    test('204 yanıtı null döner', () async {
      final api = client((_) async => jsonResponse(null, 204));

      expect(await api.post('/v1/rooms/x/leave'), isNull);
    });
  });

  group('hata eşleme', () {
    test('standart hata gövdesi → server, code ve request_id', () async {
      final api = client((_) async => jsonResponse(apiError('ROOM_FULL'), 409));

      final e = await expectApiError(api.post('/v1/rooms/join'));

      expect(e.kind, ApiErrorKind.server);
      expect(e.statusCode, 409);
      expect(e.code, 'ROOM_FULL');
      expect(e.requestId, 'req-1');
    });

    test('VALIDATION_FAILED alan hataları okunur', () async {
      final api = client(
        (_) async => jsonResponse(
          apiError(
            'VALIDATION_FAILED',
            details: {
              'fields': {'nickname': 'too_short'},
            },
          ),
          400,
        ),
      );

      final e = await expectApiError(api.post('/v1/sessions/guest'));

      expect(e.code, 'VALIDATION_FAILED');
      expect(e.fieldErrors, {'nickname': 'too_short'});
    });

    test('401 → unauthorized ve geri çağrı', () async {
      final api = client(
        (_) async => jsonResponse(apiError('UNAUTHORIZED'), 401),
      );

      final e = await expectApiError(api.get('/v1/me'));

      expect(e.kind, ApiErrorKind.unauthorized);
      expect(unauthorizedCalls, 1);
    });

    test('426 → upgradeRequired ve geri çağrı', () async {
      final api = client(
        (_) async => jsonResponse(apiError('UPGRADE_REQUIRED'), 426),
      );

      final e = await expectApiError(api.get('/v1/me'));

      expect(e.kind, ApiErrorKind.upgradeRequired);
      expect(upgradeCalls, 1);
    });

    test('429 → rateLimited ve Retry-After', () async {
      final api = client(
        (_) async => jsonResponse(
          apiError('RATE_LIMITED'),
          429,
          headers: {
            'retry-after': ['7'],
          },
        ),
      );

      final e = await expectApiError(api.post('/v1/rooms/x/messages'));

      expect(e.kind, ApiErrorKind.rateLimited);
      expect(e.retryAfter, const Duration(seconds: 7));
    });

    test('standart olmayan hata gövdesi → invalidResponse', () async {
      final api = client((_) async => jsonResponse({'oops': 1}, 502));

      final e = await expectApiError(api.get('/v1/me'));

      expect(e.kind, ApiErrorKind.invalidResponse);
      expect(e.statusCode, 502);
      expect(e.code, isNull);
    });

    test('zaman aşımı → timeout', () async {
      final api = client(
        (o) async => throw DioException.receiveTimeout(
          timeout: const Duration(seconds: 15),
          requestOptions: o,
        ),
      );

      final e = await expectApiError(api.get('/v1/me'));

      expect(e.kind, ApiErrorKind.timeout);
    });

    test('bağlantı hatası → network', () async {
      final api = client(
        (o) async => throw DioException.connectionError(
          requestOptions: o,
          reason: 'no route',
        ),
      );

      final e = await expectApiError(api.get('/v1/me'));

      expect(e.kind, ApiErrorKind.network);
    });
  });
}
