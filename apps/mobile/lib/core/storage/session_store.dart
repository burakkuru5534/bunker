import 'dart:convert';

import 'package:flutter/foundation.dart';

import 'package:siginak/core/storage/secure_store.dart';

/// Cihazda saklanan misafir oturumu (docs/security.md §1).
@immutable
class StoredSession {
  const StoredSession({
    required this.token,
    required this.expiresAt,
    required this.userId,
    required this.nickname,
  });

  final String token;
  final DateTime expiresAt;
  final String userId;
  final String nickname;

  bool isExpired(DateTime now) => !now.isBefore(expiresAt);

  StoredSession copyWith({
    String? token,
    DateTime? expiresAt,
    String? nickname,
  }) => StoredSession(
    token: token ?? this.token,
    expiresAt: expiresAt ?? this.expiresAt,
    userId: userId,
    nickname: nickname ?? this.nickname,
  );

  Map<String, Object?> toJson() => {
    'token': token,
    'expires_at': expiresAt.toUtc().toIso8601String(),
    'user_id': userId,
    'nickname': nickname,
  };

  /// Bozuk veya eksik veride `null`.
  static StoredSession? tryFromJson(Object? json) {
    if (json is! Map<String, Object?>) return null;
    final token = json['token'];
    final expires = json['expires_at'];
    final userId = json['user_id'];
    final nickname = json['nickname'];
    if (token is! String || token.isEmpty) return null;
    if (expires is! String || userId is! String || nickname is! String) {
      return null;
    }
    final expiresAt = DateTime.tryParse(expires);
    if (expiresAt == null) return null;
    return StoredSession(
      token: token,
      expiresAt: expiresAt,
      userId: userId,
      nickname: nickname,
    );
  }

  @override
  bool operator ==(Object other) =>
      other is StoredSession &&
      other.token == token &&
      other.expiresAt == expiresAt &&
      other.userId == userId &&
      other.nickname == nickname;

  @override
  int get hashCode => Object.hash(token, expiresAt, userId, nickname);

  /// Token loglara / hata mesajlarına sızmasın.
  @override
  String toString() => 'StoredSession(userId: $userId, expiresAt: $expiresAt)';
}

/// Oturumu güvenli depoda saklar; bellekte önbellekler.
///
/// - Bozuk veri bulunursa silinir ve oturum yok sayılır.
/// - Süresi dolmuş oturum okunurken silinir.
/// - Token her istekte okunduğu için ilk okumadan sonra bellekten döner.
class SessionStore {
  SessionStore(this._store, {DateTime Function()? clock})
    : _clock = clock ?? DateTime.now;

  /// Depolama anahtarı; biçim değişirse sürüm artırılır.
  static const storageKey = 'session.v1';

  final SecureStore _store;
  final DateTime Function() _clock;

  StoredSession? _cached;
  bool _loaded = false;

  Future<StoredSession?> read() async {
    if (!_loaded) {
      _cached = await _load();
      _loaded = true;
    }
    final session = _cached;
    if (session != null && session.isExpired(_clock())) {
      await clear();
      return null;
    }
    return session;
  }

  /// Geçerli token (oturum yoksa veya süresi dolduysa `null`).
  Future<String?> readToken() async => (await read())?.token;

  Future<void> save(StoredSession session) async {
    await _store.write(storageKey, jsonEncode(session.toJson()));
    _cached = session;
    _loaded = true;
  }

  Future<void> clear() async {
    _cached = null;
    _loaded = true;
    await _store.delete(storageKey);
  }

  Future<StoredSession?> _load() async {
    final raw = await _store.read(storageKey);
    if (raw == null) return null;
    StoredSession? session;
    try {
      session = StoredSession.tryFromJson(jsonDecode(raw));
    } on FormatException {
      session = null;
    }
    if (session == null) await _store.delete(storageKey);
    return session;
  }
}
