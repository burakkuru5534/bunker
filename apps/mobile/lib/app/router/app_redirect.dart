import 'package:siginak/app/router/app_routes.dart';
import 'package:siginak/features/session/session.dart';

/// Açılış ve oturum durumuna göre yönlendirme kararı (saf fonksiyon).
///
/// `null` → istenen adrese git; aksi halde döndürülen adrese yönlendir.
/// Kurallar docs/ui/screen-map.md §2 ile aynıdır:
/// 1. Güncelleme gerekiyorsa her şey `/update`'e kilitlenir.
/// 2. Oturum bilinmiyorsa açılış ekranında beklenir.
/// 3. Oturum yoksa yalnızca `/welcome` açıktır.
/// 4. Oturum varken `/` ve `/welcome` → aktif oda varsa odaya, yoksa ana ekrana.
String? appRedirect({
  required Uri uri,
  required SessionState session,
  required bool updateRequired,
}) {
  final path = uri.path;

  if (updateRequired) {
    return path == AppRoutes.update ? null : AppRoutes.update;
  }
  if (path == AppRoutes.update) {
    return AppRoutes.splash;
  }

  switch (session.status) {
    case SessionStatus.unknown:
      return path == AppRoutes.splash ? null : AppRoutes.splash;
    case SessionStatus.unauthenticated:
      return path == AppRoutes.welcome ? null : AppRoutes.welcome;
    case SessionStatus.authenticated:
      if (path == AppRoutes.splash || path == AppRoutes.welcome) {
        final roomId = session.activeRoomId;
        return roomId != null ? AppRoutes.room(roomId) : AppRoutes.home;
      }
      return null;
  }
}
