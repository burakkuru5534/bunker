import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/config/app_config_provider.dart';
import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/network/api_client_provider.dart';

import '../../helpers/pump_app.dart';

void main() {
  test('yapılandırmayla istemci oluşturulur', () {
    final container = ProviderContainer(
      overrides: [appConfigProvider.overrideWithValue(testConfig())],
    );
    addTearDown(container.dispose);

    expect(container.read(apiClientProvider), isA<ApiClient>());
  });
}
