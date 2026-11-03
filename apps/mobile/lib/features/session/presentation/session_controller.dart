import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/storage/session_store.dart';
import 'package:siginak/core/storage/storage_providers.dart';

/// Oturumun yönlendirme için önemli olan durumu.
enum SessionStatus {
  /// Açılışta oturum henüz kontrol edilmedi.
  unknown,

  /// Geçerli oturum yok → `/welcome`.
  unauthenticated,

  /// Geçerli misafir oturumu var.
  authenticated,
}

@immutable
class SessionState {
  const SessionState({
    required this.status,
    this.userId,
    this.nickname,
    this.activeRoomId,
  });

  const SessionState.unknown() : this(status: SessionStatus.unknown);

  const SessionState.unauthenticated()
    : this(status: SessionStatus.unauthenticated);

  final SessionStatus status;
  final String? userId;
  final String? nickname;

  /// Kullanıcının içinde bulunduğu aktif oda (`GET /v1/me`), yoksa `null`.
  final String? activeRoomId;

  @override
  bool operator ==(Object other) =>
      other is SessionState &&
      other.status == status &&
      other.userId == userId &&
      other.nickname == nickname &&
      other.activeRoomId == activeRoomId;

  @override
  int get hashCode => Object.hash(status, userId, nickname, activeRoomId);
}

/// Oturum durumu.
///
/// Açılışta `unknown` ile başlar ve güvenli depodaki oturumu okur:
/// geçerli oturum varsa `authenticated`, yoksa `unauthenticated` olur.
/// Backend ile oturum oluşturma ve `GET /v1/me` (aktif oda) AUTH-002'de
/// eklenecektir.
class SessionController extends Notifier<SessionState> {
  @override
  SessionState build() {
    unawaited(_restore());
    return const SessionState.unknown();
  }

  SessionStore get _store => ref.read(sessionStoreProvider);

  Future<void> _restore() async {
    StoredSession? session;
    try {
      session = await _store.read();
    } on Object {
      // Keystore okunamazsa (ör. cihaz değişti) oturum yok sayılır.
      session = null;
    }
    if (!ref.mounted) return;
    state = session == null
        ? const SessionState.unauthenticated()
        : _authenticated(session);
  }

  /// Sunucudan alınan oturumu saklar ve oturumu açar.
  Future<void> signIn(StoredSession session) async {
    await _store.save(session);
    state = _authenticated(session);
  }

  /// Oturumu kapatır ve cihazdan siler (ör. 401, hesap silme).
  Future<void> signOut() async {
    await _store.clear();
    state = const SessionState.unauthenticated();
  }

  /// GEÇİCİ: Backend bağlanana kadar (AUTH-002) misafir girişini taklit eder.
  /// Yerel bir sahte oturum saklar; sunucuya gönderilse geçersiz sayılır.
  Future<void> signInAsGuestPlaceholder() => signIn(
    StoredSession(
      token: 'local-placeholder',
      expiresAt: DateTime.now().add(const Duration(days: 90)),
      userId: 'local',
      nickname: 'Misafir',
    ),
  );

  SessionState _authenticated(StoredSession s) => SessionState(
    status: SessionStatus.authenticated,
    userId: s.userId,
    nickname: s.nickname,
  );
}

final sessionControllerProvider =
    NotifierProvider<SessionController, SessionState>(SessionController.new);
