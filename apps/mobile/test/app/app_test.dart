import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/app/app.dart';
import 'package:siginak/features/home/presentation/home_screen.dart';

import '../helpers/pump_app.dart';

void main() {
  testWidgets('Uygulama ana ekranla açılır', (tester) async {
    await pumpWithProviders(
      tester,
      const SiginakApp(),
      wrapInMaterialApp: false,
    );

    expect(find.byType(HomeScreen), findsOneWidget);
  });
}
