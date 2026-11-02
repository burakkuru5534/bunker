import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

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
  const SessionState({required this.status, this.activeRoomId});

  const SessionState.unknown() : this(status: SessionStatus.unknown);

  const SessionState.unauthenticated()
    : this(status: SessionStatus.unauthenticated);

  final SessionStatus status;

  /// Kullanıcının içinde bulunduğu aktif oda (`GET /v1/me`), yoksa `null`.
  final String? activeRoomId;

  @override
  bool operator ==(Object other) =>
      other is SessionState &&
      other.status == status &&
      other.activeRoomId == activeRoomId;

  @override
  int get hashCode => Object.hash(status, activeRoomId);
}

/// Oturum durumu.
///
/// Şimdilik bellek içi bir iskelettir: uygulama her açılışta oturumsuz
/// başlar. Güvenli depolama (MOB-010) ve backend oturumu (AUTH-002)
/// eklendiğinde açılışta `unknown` ile başlayıp gerçek kontrol yapılacaktır.
class SessionController extends Notifier<SessionState> {
  @override
  SessionState build() => const SessionState.unauthenticated();

  /// GEÇİCİ: Backend bağlanana kadar misafir girişini taklit eder.
  void signInAsGuestPlaceholder() {
    state = const SessionState(status: SessionStatus.authenticated);
  }
}

final sessionControllerProvider =
    NotifierProvider<SessionController, SessionState>(SessionController.new);
