import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/features/home/presentation/home_screen.dart';

void main() {
  Future<void> pump(WidgetTester tester, AppConfig config) {
    return tester.pumpWidget(MaterialApp(home: HomeScreen(config: config)));
  }

  testWidgets('Başlık ve local ortam etiketi görünür', (tester) async {
    await pump(tester, AppConfig.fromEnvironment());

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
    await pump(tester, prod);

    expect(find.textContaining('Ortam:'), findsNothing);
  });
}
