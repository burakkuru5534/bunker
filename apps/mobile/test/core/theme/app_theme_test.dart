import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/theme/app_colors.dart';
import 'package:siginak/core/theme/app_theme.dart';
import 'package:siginak/core/theme/game_theme.dart';

void main() {
  final theme = buildAppTheme();

  test('koyu Material 3 tema ve marka renkleri', () {
    expect(theme.useMaterial3, isTrue);
    expect(theme.brightness, Brightness.dark);
    expect(theme.colorScheme.primary, AppColors.primary);
    expect(theme.colorScheme.error, AppColors.danger);
    expect(theme.scaffoldBackgroundColor, AppColors.background);
  });

  test('GameTheme eklentisi kayıtlı', () {
    final game = theme.extension<GameTheme>();

    expect(game, isNotNull);
    expect(game!.profession.color, AppColors.categoryProfession);
    expect(game.secret.icon, Icons.lock_rounded);
    expect(game.health.label, 'Sağlık durumu');
  });

  testWidgets('context.gameTheme ve düğme yüksekliği', (tester) async {
    late GameTheme fromContext;
    await tester.pumpWidget(
      MaterialApp(
        theme: theme,
        home: Builder(
          builder: (context) {
            fromContext = context.gameTheme;
            return Scaffold(
              body: FilledButton(onPressed: () {}, child: const Text('Başla')),
            );
          },
        ),
      ),
    );

    expect(fromContext, same(GameTheme.dark));
    expect(
      tester.getSize(find.byType(FilledButton)).height,
      greaterThanOrEqualTo(52),
    );
  });

  test('avatar rengi koltuk sırasına göre döngüsel', () {
    const g = GameTheme.dark;

    expect(g.avatarColorFor(0), g.avatarColors[0]);
    expect(g.avatarColorFor(8), g.avatarColors[0]);
    expect(g.avatarColorFor(9), g.avatarColors[1]);
  });

  test('copyWith ve lerp', () {
    const g = GameTheme.dark;
    final changed = g.copyWith(danger: const Color(0xFF000000));

    expect(changed.danger, const Color(0xFF000000));
    expect(changed.profession, g.profession);
    expect(g.lerp(changed, 0.2), same(g));
    expect(g.lerp(changed, 0.8), same(changed));
  });
}
