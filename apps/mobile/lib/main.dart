import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/app/app.dart';
import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/core/config/app_config_provider.dart';
import 'package:siginak/core/theme/font_licenses.dart';

void main() {
  registerFontLicenses();
  final config = AppConfig.fromEnvironment();
  runApp(
    ProviderScope(
      overrides: [appConfigProvider.overrideWithValue(config)],
      child: const SiginakApp(),
    ),
  );
}
