import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/storage/session_store.dart';
import 'package:siginak/core/storage/storage_providers.dart';
import 'package:siginak/features/session/data/session_repository.dart';

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
    this.restoreError,
    this.needsNickname = false,
  });

  const SessionState.unknown() : this(status: SessionStatus.unknown);

  /// Saklı oturum sunucuyla doğrulanamadı (ağ yok, sunucu hatası…);
  /// açılış ekranı hatayı ve "Tekrar dene"yi gösterir.
  const SessionState.restoreFailed(Object error)
    : this(status: SessionStatus.unknown, restoreError: error);

  const SessionState.unauthenticated()
    : this(status: SessionStatus.unauthenticated);

  final SessionStatus status;
  final String? userId;
  final String? nickname;

  /// Kullanıcının içinde bulunduğu aktif oda (`GET /v1/me`), yoksa `null`.
  final String? activeRoomId;

  /// Yalnızca `unknown` durumunda: doğrulama hatası (yoksa `null`).
  final Object? restoreError;

  /// Kullanıcı adı onaylanmalı (`GET /v1/me` `needs_nickname`).
  final bool needsNickname;

  @override
  bool operator ==(Object other) =>
      other is SessionState &&
      other.status == status &&
      other.userId == userId &&
      other.nickname == nickname &&
      other.activeRoomId == activeRoomId &&
      other.restoreError == restoreError &&
      other.needsNickname == needsNickname;

  @override
  int get hashCode => Object.hash(
    status,
    userId,
    nickname,
    activeRoomId,
    restoreError,
    needsNickname,
  );
}

/// Oturum durumu.
///
/// Açılışta `unknown` ile başlar. Güvenli depoda oturum yoksa (veya süresi
/// dolmuşsa) `unauthenticated` olur. Varsa `GET /v1/me` ile sunucuda
/// doğrulanır: geçerliyse `authenticated` (+ aktif oda, docs/ui/screen-map.md
/// §2), 401'de oturum silinir, ağ/sunucu hatasında `restoreFailed` ile
/// açılış ekranında beklenir. Bitişine [refreshWindow]'dan az kalan token
/// sessizce yenilenir.
class SessionController extends Notifier<SessionState> {
  /// Bitişine bundan az kalan token açılışta yenilenir.
  static const refreshWindow = Duration(days: 14);

  @override
  SessionState build() {
    unawaited(_restore());
    return const SessionState.unknown();
  }

  SessionStore get _store => ref.read(sessionStoreProvider);
  SessionRepository get _repo => ref.read(sessionRepositoryProvider);

  Future<void> _restore() async {
    StoredSession? session;
    try {
      session = await _store.read();
    } on Object {
      // Keystore okunamazsa (ör. cihaz değişti) oturum yok sayılır.
      session = null;
    }
    if (!ref.mounted) return;
    if (session == null) {
      state = const SessionState.unauthenticated();
      return;
    }

    final MeInfo me;
    try {
      me = await _repo.fetchMe();
    } on ApiException catch (e) {
      if (!ref.mounted) return;
      switch (e.kind) {
        case ApiErrorKind.unauthorized:
          await signOut();
        case ApiErrorKind.upgradeRequired:
          // Yönlendirme /update'e kilitlenir; oturum korunur.
          state = const SessionState.unknown();
        default:
          state = SessionState.restoreFailed(e);
      }
      return;
    }
    if (!ref.mounted) return;

    if (me.nickname != session.nickname) {
      session = session.copyWith(nickname: me.nickname);
      await _store.save(session);
      if (!ref.mounted) return;
    }
    state = _authenticated(
      session,
      activeRoomId: me.activeRoomId,
      needsNickname: me.needsNickname,
    );
    await _refreshIfNeeded(session);
  }

  Future<void> _refreshIfNeeded(StoredSession session) async {
    final left = session.expiresAt.difference(DateTime.now());
    if (left > refreshWindow) return;
    try {
      final t = await _repo.refresh();
      await _store.save(
        session.copyWith(token: t.token, expiresAt: t.expiresAt),
      );
    } on ApiException {
      // Yenileme başarısızsa mevcut token bitene kadar kullanılır; 401 ise
      // istemci zaten oturumu kapatır.
    }
  }

  /// Doğrulama hatasından sonra yeniden dener (açılış ekranı).
  Future<void> retryRestore() async {
    state = const SessionState.unknown();
    await _restore();
  }

  /// Misafir hesap açar ve oturumu başlatır. Hata [ApiException] olarak
  /// fırlatılır (ekran alan hatasını / mesajı gösterir).
  Future<void> signInAsGuest(String nickname) async {
    final session = await _repo.createGuest(nickname);
    await signIn(session);
  }

  /// Takma adı değiştirir; hata [ApiException] olarak fırlatılır
  /// (oyun sürerken `INVALID_PHASE`, kurallar için `NICKNAME_INVALID`).
  Future<void> changeNickname(String nickname) async {
    final saved = await _repo.updateNickname(nickname);
    final session = await _store.read();
    if (session != null) await _store.save(session.copyWith(nickname: saved));
    if (!ref.mounted || state.status != SessionStatus.authenticated) return;
    state = SessionState(
      status: SessionStatus.authenticated,
      userId: state.userId,
      nickname: saved,
      activeRoomId: state.activeRoomId,
    );
  }

  /// Kullanıcı adını onaylar (adı değişmese de sunucuya kaydedilir).
  Future<void> confirmNickname(String nickname) => changeNickname(nickname);

  /// Kullanıcının aktif odası değişti (oda kuruldu, katılındı, ayrılındı).
  void setActiveRoom(String? roomId) {
    if (state.status != SessionStatus.authenticated) return;
    if (state.activeRoomId == roomId) return;
    state = SessionState(
      status: SessionStatus.authenticated,
      userId: state.userId,
      nickname: state.nickname,
      activeRoomId: roomId,
      needsNickname: state.needsNickname,
    );
  }

  /// Sunucudan alınan oturumu saklar ve oturumu açar.
  Future<void> signIn(StoredSession session) async {
    await _store.save(session);
    state = _authenticated(session);
  }

  /// Hesabı sunucuda siler, sonra cihazdaki oturumu kapatır. Hata
  /// [ApiException] olarak fırlatılır (oturum korunur).
  Future<void> deleteAccount() async {
    await _repo.deleteAccount();
    await signOut();
  }

  /// Oturumu kapatır ve cihazdan siler (ör. 401, hesap silme).
  Future<void> signOut() async {
    await _store.clear();
    if (!ref.mounted) return;
    state = const SessionState.unauthenticated();
  }

  SessionState _authenticated(
    StoredSession s, {
    String? activeRoomId,
    bool needsNickname = false,
  }) => SessionState(
    status: SessionStatus.authenticated,
    userId: s.userId,
    nickname: s.nickname,
    activeRoomId: activeRoomId,
    needsNickname: needsNickname,
  );
}

final sessionControllerProvider =
    NotifierProvider<SessionController, SessionState>(SessionController.new);
