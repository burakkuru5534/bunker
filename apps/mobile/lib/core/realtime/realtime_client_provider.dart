import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/config/app_config_provider.dart';
import 'package:siginak/core/config/app_version.dart';
import 'package:siginak/core/network/auth_token_provider.dart';
import 'package:siginak/core/network/update_required_provider.dart';
import 'package:siginak/core/realtime/realtime_client.dart';

/// Uygulama boyu tek WebSocket istemcisi.
final realtimeClientProvider = Provider<RealtimeClient>((ref) {
  final config = ref.watch(appConfigProvider);
  final client = RealtimeClient(
    wsBaseUrl: config.wsBaseUrl,
    appVersion: kAppVersion,
    readToken: ref.watch(authTokenReaderProvider),
    onUnauthorized: ref.watch(unauthorizedHandlerProvider),
    onUpgradeRequired: () =>
        ref.read(updateRequiredProvider.notifier).markRequired(),
  );
  ref.onDispose(client.dispose);
  return client;
});

/// Bağlantı durumu (bağlantı şeridi vb. için).
final realtimeStateProvider = StreamProvider<RealtimeState>((ref) {
  final client = ref.watch(realtimeClientProvider);
  return client.states;
});
