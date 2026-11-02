import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/app/app.dart';
import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/features/home/presentation/home_screen.dart';

void main() {
  testWidgets('Uygulama ana ekranla açılır', (tester) async {
    await tester.pumpWidget(SiginakApp(config: AppConfig.fromEnvironment()));

    expect(find.byType(HomeScreen), findsOneWidget);
  });
}
