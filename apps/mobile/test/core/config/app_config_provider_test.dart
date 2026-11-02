import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/core/config/app_config_provider.dart';

void main() {
  test('override edilmeden okunursa hata verir', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);

    expect(() => container.read(appConfigProvider), throwsA(anything));
  });

  test('override edilen yapılandırmayı döndürür', () {
    final config = AppConfig.fromValues(
      environment: 'test',
      apiBaseUrl: 'https://api.example.com',
      wsBaseUrl: 'wss://api.example.com',
    );
    final container = ProviderContainer(
      overrides: [appConfigProvider.overrideWithValue(config)],
    );
    addTearDown(container.dispose);

    expect(container.read(appConfigProvider), same(config));
  });
}
