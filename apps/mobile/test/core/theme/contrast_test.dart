import 'dart:math' as math;

import 'package:flutter/painting.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/theme/app_colors.dart';

/// WCAG 2.1 bağıl parlaklık.
double _luminance(Color c) {
  double ch(double v) =>
      v <= 0.03928 ? v / 12.92 : math.pow((v + 0.055) / 1.055, 2.4).toDouble();
  return 0.2126 * ch(c.r) + 0.7152 * ch(c.g) + 0.0722 * ch(c.b);
}

double contrast(Color a, Color b) {
  final la = _luminance(a);
  final lb = _luminance(b);
  return (math.max(la, lb) + 0.05) / (math.min(la, lb) + 0.05);
}

void main() {
  const normalText = 4.5;

  group('metin kontrastı ≥ 4.5:1 (docs/ui/brand.md §3)', () {
    final foregrounds = {
      'textPrimary': AppColors.textPrimary,
      'textSecondary': AppColors.textSecondary,
      'primary': AppColors.primary,
      'secondary': AppColors.secondary,
      'success': AppColors.success,
      'categoryProfession': AppColors.categoryProfession,
      'categoryHealth': AppColors.categoryHealth,
      'categoryHobby': AppColors.categoryHobby,
      'categoryItem': AppColors.categoryItem,
      'categorySecret': AppColors.categorySecret,
      'danger': AppColors.danger,
    };
    final backgrounds = {
      'background': AppColors.background,
      'surface': AppColors.surface,
      'surfaceHigh': AppColors.surfaceHigh,
    };

    for (final fg in foregrounds.entries) {
      for (final bg in backgrounds.entries) {
        test('${fg.key} / ${bg.key}', () {
          expect(
            contrast(fg.value, bg.value),
            greaterThanOrEqualTo(normalText),
          );
        });
      }
    }
  });

  test('düğme metinleri kendi arka planlarında okunur', () {
    expect(
      contrast(AppColors.onPrimary, AppColors.primary),
      greaterThanOrEqualTo(normalText),
    );
    expect(
      contrast(AppColors.onDanger, AppColors.danger),
      greaterThanOrEqualTo(normalText),
    );
    expect(
      contrast(AppColors.onSecondary, AppColors.secondary),
      greaterThanOrEqualTo(normalText),
    );
  });

  test('avatar baş harfleri tüm avatar renklerinde okunur', () {
    for (final c in AppColors.avatarPalette) {
      expect(
        contrast(AppColors.background, c),
        greaterThanOrEqualTo(normalText),
        reason: c.toString(),
      );
    }
  });
}
