import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

/// En küçük hedef ekran (docs/ui/design-system.md §5): 320×568 dp.
const smallScreen = Size(320, 568);

/// Görünümü [size] dp'ye ve sistem yazı ölçeğini [textScale]'e ayarlar
/// (docs/ui/brand.md: %200'e kadar düzen bozulmaz). Test bitince geri alınır.
///
/// Taşma (RenderFlex overflowed) çizim sırasında hata olarak raporlanır ve
/// testi kendiliğinden başarısız yapar.
void useSmallScreen(
  WidgetTester tester, {
  Size size = smallScreen,
  double textScale = 2,
}) {
  tester.view
    ..physicalSize = size * 3
    ..devicePixelRatio = 3;
  tester.platformDispatcher.textScaleFactorTestValue = textScale;
  addTearDown(tester.view.reset);
  addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
}

/// Dokunma hedefi ve etiket kuralları (Android 48 dp; etiketli hedefler).
Future<void> expectAccessibleTapTargets(WidgetTester tester) async {
  await expectLater(tester, meetsGuideline(androidTapTargetGuideline));
  await expectLater(tester, meetsGuideline(labeledTapTargetGuideline));
}
