import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/storage/session_store.dart';

/// `GET /v1/me` yanıtı.
@immutable
class MeInfo {
  const MeInfo({
    required this.id,
    required this.nickname,
    required this.activeRoomId,
    this.needsNickname = false,
  });

  final String id;
  final String nickname;

  /// Kullanıcının içinde bulunduğu oda; yoksa `null`.
  final String? activeRoomId;

  /// Benzersiz kullanıcı adı henüz onaylanmadı (eski hesap); ana ekrandan
  /// önce adını onaylar/seçer.
  final bool needsNickname;
}

/// Yenilenen token (`POST /v1/sessions/refresh`).
typedef RefreshedToken = ({String token, DateTime expiresAt});

/// Oturum uç noktaları (docs/api.md "Oturum ve kullanıcı").
class SessionRepository {
  SessionRepository(this._api);

  final ApiClient _api;

  /// Misafir hesap açar: `POST /v1/sessions/guest`.
  Future<StoredSession> createGuest(String nickname) async {
    final json = _object(
      await _api.post('/v1/sessions/guest', body: {'nickname': nickname}),
    );
    final user = _object(json['user']);
    return StoredSession(
      token: _string(json, 'token'),
      expiresAt: _time(json, 'expires_at'),
      userId: _string(user, 'id'),
      nickname: _string(user, 'nickname'),
    );
  }

  /// `GET /v1/me`.
  Future<MeInfo> fetchMe() async {
    final json = _object(await _api.get('/v1/me'));
    final room = json['active_room_id'];
    if (room != null && room is! String) throw _invalid('active_room_id');
    return MeInfo(
      id: _string(json, 'id'),
      nickname: _string(json, 'nickname'),
      activeRoomId: room as String?,
      needsNickname: json['needs_nickname'] == true,
    );
  }

  /// Takma adı değiştirir (`PATCH /v1/me`); sunucunun kaydettiği adı döndürür.
  Future<String> updateNickname(String nickname) async {
    final json = _object(
      await _api.patch('/v1/me', body: {'nickname': nickname}),
    );
    return _string(json, 'nickname');
  }

  /// `POST /v1/sessions/refresh`: eski token iptal olur.
  /// Hesabı siler (`DELETE /v1/me`; geri alınamaz).
  Future<void> deleteAccount() async {
    await _api.delete('/v1/me');
  }

  Future<RefreshedToken> refresh() async {
    final json = _object(await _api.post('/v1/sessions/refresh', body: {}));
    return (
      token: _string(json, 'token'),
      expiresAt: _time(json, 'expires_at'),
    );
  }

  static Map<String, Object?> _object(Object? v) {
    if (v is Map<String, Object?>) return v;
    throw _invalid('nesne bekleniyordu');
  }

  static String _string(Map<String, Object?> json, String key) {
    final v = json[key];
    if (v is String && v.isNotEmpty) return v;
    throw _invalid(key);
  }

  static DateTime _time(Map<String, Object?> json, String key) {
    final t = DateTime.tryParse(_string(json, key));
    if (t == null) throw _invalid(key);
    return t;
  }

  static ApiException _invalid(String what) => ApiException(
    kind: ApiErrorKind.invalidResponse,
    debugMessage: 'oturum yanıtı: $what',
  );
}

final sessionRepositoryProvider = Provider<SessionRepository>(
  (ref) => SessionRepository(ref.watch(apiClientProvider)),
);
