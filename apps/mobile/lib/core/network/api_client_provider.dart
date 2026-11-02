import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/config/app_config_provider.dart';
import 'package:siginak/core/config/app_version.dart';
import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/network/auth_token_provider.dart';
import 'package:siginak/core/network/update_required_provider.dart';

/// Uygulama boyu tek REST istemcisi.
final apiClientProvider = Provider<ApiClient>((ref) {
  final config = ref.watch(appConfigProvider);
  final client = ApiClient(
    baseUrl: config.apiBaseUrl,
    appVersion: kAppVersion,
    readToken: ref.watch(authTokenReaderProvider),
    onUnauthorized: ref.watch(unauthorizedHandlerProvider),
    onUpgradeRequired: () =>
        ref.read(updateRequiredProvider.notifier).markRequired(),
  );
  ref.onDispose(client.close);
  return client;
});
