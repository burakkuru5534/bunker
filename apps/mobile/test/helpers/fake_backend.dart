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
        'needs_nickname': needsNickname,
      }, 200),
    );
    on('PATCH', '/v1/me', (r) async {
      nickname = (r.data as Map<String, Object?>)['nickname']! as String;
      needsNickname = false;
      return jsonResponse({
        'id': 'u-guest',
        'nickname': nickname,
        'needs_nickname': false,
      }, 200);
    });
    on('POST', '/v1/rooms', (r) async {
      activeRoomId = createdRoomId;
      return jsonResponse(roomJson(id: createdRoomId), 201);
    });
    on('POST', '/v1/rooms/join', (r) async {
      activeRoomId = joinedRoomId;
      final code = (r.data as Map<String, Object?>)['invite_code']! as String;
      return jsonResponse(roomJson(id: joinedRoomId, inviteCode: code), 200);
    });
    on(
      'GET',
      '/v1/inbox',
      (r) async =>
          jsonResponse({'invites': <Object?>[], 'incoming_requests': 0}, 200),
    );
    on(
      'GET',
      '/v1/friends',
      (r) async => jsonResponse({
        'friends': <Object?>[],
        'incoming': <Object?>[],
        'outgoing': <Object?>[],
      }, 200),
    );
    _defaults.addAll(_routes);
  }

  String nickname = 'Ayşe';

  /// Benzersiz ad gelmeden önce açılmış hesap (`GET /v1/me` → onay ekranı).
  bool needsNickname = false;

  /// `POST /v1/rooms` ve `POST /v1/rooms/join` yanıtlarındaki oda kimlikleri.
  static const createdRoomId = '11111111-1111-4111-8111-111111111111';
  static const joinedRoomId = '22222222-2222-4222-8222-222222222222';
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

/// docs/api.md biçiminde `RoomView`.
Map<String, Object?> roomJson({
  String id = FakeBackend.createdRoomId,
  String inviteCode = 'K7M2QX',
  String status = 'lobby',
  String hostUserId = 'u-guest',
  int capacity = 2,
  int version = 0,
  List<Map<String, Object?>>? members,
}) => {
  'id': id,
  'invite_code': inviteCode,
  'status': status,
  'host_user_id': hostUserId,
  'capacity': capacity,
  'current_game_id': null,
  'version': version,
  'members':
      members ??
      [
        {
          'user_id': hostUserId,
          'nickname': 'Ayşe',
          'role': 'player',
          'is_ready': false,
          'connected': false,
        },
      ],
};
