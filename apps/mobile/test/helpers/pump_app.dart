import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/misc.dart' show Override;
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/core/config/app_config_provider.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/storage_providers.dart';
import 'package:siginak/core/theme/app_theme.dart';
import 'package:siginak/features/game/presentation/intro/intro_audio.dart';

import 'fake_intro_audio.dart';
import 'pump_themed.dart' show testLocalizationsDelegates;

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
  SecureStore? secureStore,
  IntroAudio Function()? introAudio,
  bool wrapInMaterialApp = true,
}) {
  return tester.pumpWidget(
    ProviderScope(
      overrides: [
        appConfigProvider.overrideWithValue(config ?? testConfig()),
        // Testlerde platform eklentisi yok: bellek içi güvenli depo.
        secureStoreProvider.overrideWithValue(
          secureStore ?? InMemorySecureStore(),
        ),
        // Testlerde ses eklentisi yok: tanıtım sesi sessiz sahte.
        introAudioFactoryProvider.overrideWithValue(
          introAudio ?? FakeIntroAudio.new,
        ),
        ...overrides,
      ],
      child: wrapInMaterialApp
          ? MaterialApp(
              theme: buildAppTheme(),
              locale: const Locale('tr'),
              supportedLocales: AppLocalizations.supportedLocales,
              localizationsDelegates: testLocalizationsDelegates,
              home: child,
            )
          : child,
    ),
  );
}
