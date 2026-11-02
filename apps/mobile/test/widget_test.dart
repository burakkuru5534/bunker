import 'package:flutter_test/flutter_test.dart';
import 'package:siginak/main.dart';

void main() {
  testWidgets('Uygulama açılır ve başlık görünür', (tester) async {
    await tester.pumpWidget(const SiginakApp());

    expect(find.text('SIĞINAK'), findsOneWidget);
    expect(find.text('Son Kalanlar'), findsOneWidget);
  });
}
