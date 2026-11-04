import 'package:dio/dio.dart';

import 'fake_http_adapter.dart';

/// Yanıt üretici. Tanımsız yollar 404 NOT_FOUND döner.
typedef FakeRoute = Future<ResponseBody> Function(RequestOptions request);

/// Yöntem + yol eşlemeli sahte sunucu (ağa çıkmaz).
///
/// Varsayılan olarak misafir oturumu açma ve `GET /v1/me` başarılıdır;
/// testler [on] ile tek tek yanıtları değiştirir.
class FakeBackend {
  FakeBackend({this.activeRoomId}) {
    on('POST', '/v1/sessions/guest', (r) async {
      final body = r.data as Map<String, Object?>;
      nickname = body['nickname']! as String;
      return jsonResponse({
        'token': 'tok-guest',
        'expires_at': DateTime.now()
            .add(const Duration(days: 90))
            .toUtc()
            .toIso8601String(),
        'user': {'id': 'u-guest', 'nickname': nickname},
      }, 201);
    });
    on(
      'GET',
      '/v1/me',
      (r) async => jsonResponse({
        'id': 'u-guest',
        'nickname': nickname,
        'active_room_id': activeRoomId,
      }, 200),
    );
    _defaults.addAll(_routes);
  }

  String nickname = 'Ayşe';
  String? activeRoomId;

  final Map<String, FakeRoute> _routes = {};
  final Map<String, FakeRoute> _defaults = {};
  late final adapter = FakeHttpAdapter(_handle);

  /// İstenen yollar sırasıyla (`GET /v1/me`).
  List<String> get calls => [
    for (final r in adapter.requests) '${r.method} ${r.path}',
  ];

  void on(String method, String path, FakeRoute route) =>
      _routes['$method $path'] = route;

  /// Yolu varsayılan (başarılı) yanıtına döndürür.
  void restore(String method, String path) =>
      _routes['$method $path'] = _defaults['$method $path']!;

  Future<ResponseBody> _handle(RequestOptions r) {
    final route = _routes['${r.method} ${r.path}'];
    if (route == null) {
      return Future.value(jsonResponse(apiError('NOT_FOUND'), 404));
    }
    return route(r);
  }
}

/// Ağ hatası üreten yanıt (bağlantı kurulamadı).
Future<ResponseBody> networkFailure(RequestOptions r) => Future.error(
  DioException.connectionError(requestOptions: r, reason: 'test'),
);
