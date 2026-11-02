import 'package:flutter_test/flutter_test.dart';
import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/main.dart';

void main() {
  testWidgets('Uygulama açılır, başlık ve ortam etiketi görünür', (
    tester,
  ) async {
    await tester.pumpWidget(SiginakApp(config: AppConfig.fromEnvironment()));

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
    await tester.pumpWidget(SiginakApp(config: prod));

    expect(find.textContaining('Ortam:'), findsNothing);
  });
}
