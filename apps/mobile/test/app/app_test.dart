import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/app/app.dart';
import 'package:siginak/features/session/session.dart';

import '../helpers/pump_app.dart';

void main() {
  testWidgets('Uygulama oturumsuz açılınca hoş geldin ekranına gider', (
    tester,
  ) async {
    await pumpWithProviders(
      tester,
      const SiginakApp(),
      wrapInMaterialApp: false,
    );
    await tester.pumpAndSettle();

    expect(find.byType(WelcomeScreen), findsOneWidget);
  });
}
