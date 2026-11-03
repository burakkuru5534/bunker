import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/app/app.dart';
import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/core/config/app_config_provider.dart';
import 'package:siginak/core/error/error_reporting.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/auth_token_provider.dart';
import 'package:siginak/core/theme/font_licenses.dart';
import 'package:siginak/features/session/session.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  installGlobalErrorHandlers(
    fallbackMessage: lookupAppLocalizations(const Locale('tr'))
        .errorWidgetMessage,
  );
  registerFontLicenses();
  final config = AppConfig.fromEnvironment();
  runApp(
    ProviderScope(
      overrides: [
        appConfigProvider.overrideWithValue(config),
        // 401 / WS 4001 → oturumu kapat (core, feature'a bağımlı olamaz).
        unauthorizedHandlerProvider.overrideWith(
          (ref) =>
              () => unawaited(
                ref.read(sessionControllerProvider.notifier).signOut(),
              ),
        ),
      ],
      child: const SiginakApp(),
    ),
  );
}
