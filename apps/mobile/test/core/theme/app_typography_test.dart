import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/theme/app_typography.dart';

void main() {
  test('Türkçe büyük harf: i → İ, ı → I', () {
    expect(turkishUpperCase('sığınak'), 'SIĞINAK');
    expect(turkishUpperCase('istanbul ılık şiir'), 'İSTANBUL ILIK ŞİİR');
    expect(turkishUpperCase('elendi'), 'ELENDİ');
  });

  test('Türkçe küçük harf: I → ı, İ → i', () {
    expect(turkishLowerCase('SIĞINAK'), 'sığınak');
    expect(turkishLowerCase('İZLEYİCİ'), 'izleyici');
  });

  test('başlıklar Oswald, gövde Inter', () {
    final t = buildTextTheme(const Color(0xFFFFFFFF), const Color(0xFF888888));

    for (final s in [t.displayLarge, t.headlineMedium, t.titleLarge]) {
      expect(s!.fontFamily, AppFonts.display);
    }
    for (final s in [t.bodyLarge, t.bodyMedium, t.labelLarge, t.titleSmall]) {
      expect(s!.fontFamily, AppFonts.body);
    }
    expect(t.displayLarge!.fontSize, 40);
    expect(t.displayLarge!.fontWeight, FontWeight.w700);
    expect(t.bodyLarge!.fontSize, 16);
    expect(t.bodySmall!.color, const Color(0xFF888888));
  });
}
