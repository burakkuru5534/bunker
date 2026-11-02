import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/config/app_config.dart';

/// Uygulama yapılandırması.
///
/// Değer `main.dart` içinde `ProviderScope.overrides` ile verilir; testler
/// kendi yapılandırmalarını aynı yolla verir. Override edilmeden okunursa
/// hata fırlatır (yanlış ortamla sessizce çalışmayı önler).
final appConfigProvider = Provider<AppConfig>(
  (ref) => throw UnimplementedError(
    'appConfigProvider, ProviderScope içinde override edilmelidir.',
  ),
);
