import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/misc.dart' show Override;
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/core/config/app_config_provider.dart';

/// Testlerde kullanılan varsayılan yapılandırma (local ortam).
AppConfig testConfig() => AppConfig.fromValues(
  environment: 'local',
  apiBaseUrl: 'http://localhost:8080',
  wsBaseUrl: 'ws://localhost:8080',
);

/// Bir widget'ı ProviderScope ve MaterialApp ile sarıp çizer.
Future<void> pumpWithProviders(
  WidgetTester tester,
  Widget child, {
  AppConfig? config,
  List<Override> overrides = const [],
  bool wrapInMaterialApp = true,
}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: [
        appConfigProvider.overrideWithValue(config ?? testConfig()),
        ...overrides,
      ],
      child: wrapInMaterialApp ? MaterialApp(home: child) : child,
    ),
  );
}
