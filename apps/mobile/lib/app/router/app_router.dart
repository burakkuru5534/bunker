import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:siginak/app/router/app_redirect.dart';
import 'package:siginak/app/router/app_routes.dart';
import 'package:siginak/core/network/update_required_provider.dart';
import 'package:siginak/core/ui/not_found_screen.dart';
import 'package:siginak/features/friends/friends.dart';
import 'package:siginak/features/home/presentation/home_screen.dart';
import 'package:siginak/features/home/presentation/join_screen.dart';
import 'package:siginak/features/moderation/presentation/blocked_users_screen.dart';
import 'package:siginak/features/room/presentation/room_screen.dart';
import 'package:siginak/features/session/session.dart';
import 'package:siginak/features/settings/presentation/settings_screen.dart';
import 'package:siginak/features/startup/presentation/splash_screen.dart';
import 'package:siginak/features/startup/presentation/update_required_screen.dart';

/// Uygulamanın tek `GoRouter` örneği.
///
/// Oturum veya güncelleme durumu değişince `refreshListenable` üzerinden
/// yönlendirme yeniden değerlendirilir. Feature ekranları birbirine
/// doğrudan gitmez; gezinme geri çağrıları burada bağlanır.
final appRouterProvider = Provider<GoRouter>((ref) {
  final refresh = ValueNotifier<int>(0);
  ref
    ..listen(sessionControllerProvider, (_, _) => refresh.value++)
    ..listen(updateRequiredProvider, (_, _) => refresh.value++);

  final router = GoRouter(
    initialLocation: AppRoutes.splash,
    refreshListenable: refresh,
    redirect: (context, state) => appRedirect(
      uri: state.uri,
      session: ref.read(sessionControllerProvider),
      updateRequired: ref.read(updateRequiredProvider),
    ),
    errorBuilder: (context, state) =>
        NotFoundScreen(onGoHome: () => context.go(AppRoutes.home)),
    routes: [
      GoRoute(
        path: AppRoutes.splash,
        builder: (context, state) => const SplashScreen(),
      ),
      GoRoute(
        path: AppRoutes.welcome,
        builder: (context, state) => const WelcomeScreen(),
      ),
      GoRoute(
        path: AppRoutes.home,
        builder: (context, state) => HomeScreen(
          onJoin: () => context.push(AppRoutes.join),
          onOpenSettings: () => context.push(AppRoutes.settings),
          onOpenFriends: () => context.push(AppRoutes.friends),
          onJoinInvite: (code) => context.push(AppRoutes.joinWithCode(code)),
          // go: oda açılınca ana ekran / katılma ekranı yığından çıkar.
          onOpenRoom: (id) => context.go(AppRoutes.room(id)),
        ),
      ),
      GoRoute(
        path: AppRoutes.join,
        builder: (context, state) => JoinScreen(
          initialCode: state.uri.queryParameters['code'],
          onOpenRoom: (id) => context.go(AppRoutes.room(id)),
        ),
      ),
      GoRoute(
        path: AppRoutes.nickname,
        builder: (context, state) => const NicknameRequiredScreen(),
      ),
      GoRoute(
        path: AppRoutes.friends,
        builder: (context, state) => const FriendsScreen(),
      ),
      GoRoute(
        path: AppRoutes.roomPattern,
        builder: (context, state) {
          final roomId = state.pathParameters['roomId']!;
          if (!AppRoutes.isValidRoomId(roomId)) {
            return NotFoundScreen(onGoHome: () => context.go(AppRoutes.home));
          }
          return RoomScreen(
            roomId: roomId,
            onExit: () => context.go(AppRoutes.home),
          );
        },
      ),
      GoRoute(
        path: AppRoutes.settings,
        builder: (context, state) => SettingsScreen(
          onOpenBlockedUsers: () => context.push(AppRoutes.blockedUsers),
        ),
        routes: [
          GoRoute(
            path: 'blocks',
            builder: (context, state) => const BlockedUsersScreen(),
          ),
        ],
      ),
      GoRoute(
        path: AppRoutes.update,
        builder: (context, state) => const UpdateRequiredScreen(),
      ),
    ],
  );

  ref.onDispose(() {
    router.dispose();
    refresh.dispose();
  });
  return router;
});
