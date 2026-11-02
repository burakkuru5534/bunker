import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/features/home/presentation/home_screen.dart';

import '../../../helpers/pump_app.dart';

void main() {
  testWidgets('Başlık ve local ortam etiketi görünür', (tester) async {
    await pumpWithProviders(tester, const HomeScreen());

    expect(find.text('SIĞINAK'), findsOneWidget);
    expect(find.text('Son Kalanlar'), findsOneWidget);
    expect(find.textContaining('Ortam: local'), findsOneWidget);
  });

  testWidgets('Production ortamında ortam etiketi gösterilmez', (tester) async {
    final prod = AppConfig.fromValues(
      environment: 'prod',
      apiBaseUrl: 'https://api.example.com',
      wsBaseUrl: 'wss://api.example.com',
    );
    await pumpWithProviders(tester, const HomeScreen(), config: prod);

    expect(find.textContaining('Ortam:'), findsNothing);
  });
}
