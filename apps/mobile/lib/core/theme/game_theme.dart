import 'package:flutter/material.dart';

import 'package:siginak/core/theme/app_colors.dart';

/// Bir kart kategorisinin görsel kimliği: renk + ikon + etiket.
///
/// Renk tek başına anlam taşımaz; ikon ve etiket her zaman birlikte
/// kullanılır (docs/ui/brand.md §3).
@immutable
class CategoryStyle {
  const CategoryStyle({
    required this.color,
    required this.icon,
    required this.label,
  });

  final Color color;
  final IconData icon;
  final String label;
}

/// Oyuna özgü tema değerleri (`Theme.of(context).extension<GameTheme>()`).
@immutable
class GameTheme extends ThemeExtension<GameTheme> {
  const GameTheme({
    required this.profession,
    required this.health,
    required this.hobby,
    required this.item,
    required this.secret,
    required this.avatarColors,
    required this.onAvatar,
    required this.danger,
    required this.onDanger,
    required this.success,
    required this.hope,
  });

  /// Uygulamanın tek (koyu) teması için değerler.
  static const dark = GameTheme(
    profession: CategoryStyle(
      color: AppColors.categoryProfession,
      icon: Icons.work_rounded,
      label: 'Meslek',
    ),
    health: CategoryStyle(
      color: AppColors.categoryHealth,
      icon: Icons.medical_services_rounded,
      label: 'Sağlık durumu',
    ),
    hobby: CategoryStyle(
      color: AppColors.categoryHobby,
      icon: Icons.palette_rounded,
      label: 'Hobi / beceri',
    ),
    item: CategoryStyle(
      color: AppColors.categoryItem,
      icon: Icons.backpack_rounded,
      label: 'Eşya',
    ),
    secret: CategoryStyle(
      color: AppColors.categorySecret,
      icon: Icons.lock_rounded,
      label: 'Gizli sır',
    ),
    avatarColors: AppColors.avatarPalette,
    onAvatar: AppColors.background,
    danger: AppColors.danger,
    onDanger: AppColors.onDanger,
    success: AppColors.success,
    hope: AppColors.secondary,
  );

  final CategoryStyle profession;
  final CategoryStyle health;
  final CategoryStyle hobby;
  final CategoryStyle item;
  final CategoryStyle secret;

  /// Oyuncu avatar renk döngüsü.
  final List<Color> avatarColors;

  /// Avatar üzerindeki baş harf rengi.
  final Color onAvatar;

  /// Eleme, oylama hedefi, hata.
  final Color danger;
  final Color onDanger;

  /// Onay, bağlantı kuruldu.
  final Color success;

  /// Sığınak / umut: kazananlar, "hazır".
  final Color hope;

  /// Odadaki sırasına göre (0'dan başlar) avatar rengi.
  Color avatarColorFor(int seat) => avatarColors[seat % avatarColors.length];

  @override
  GameTheme copyWith({
    CategoryStyle? profession,
    CategoryStyle? health,
    CategoryStyle? hobby,
    CategoryStyle? item,
    CategoryStyle? secret,
    List<Color>? avatarColors,
    Color? onAvatar,
    Color? danger,
    Color? onDanger,
    Color? success,
    Color? hope,
  }) {
    return GameTheme(
      profession: profession ?? this.profession,
      health: health ?? this.health,
      hobby: hobby ?? this.hobby,
      item: item ?? this.item,
      secret: secret ?? this.secret,
      avatarColors: avatarColors ?? this.avatarColors,
      onAvatar: onAvatar ?? this.onAvatar,
      danger: danger ?? this.danger,
      onDanger: onDanger ?? this.onDanger,
      success: success ?? this.success,
      hope: hope ?? this.hope,
    );
  }

  /// Tek tema olduğu için ara değer üretilmez; yarı noktada geçiş yapılır.
  @override
  GameTheme lerp(ThemeExtension<GameTheme>? other, double t) {
    if (other is! GameTheme) return this;
    return t < 0.5 ? this : other;
  }
}

/// `context.gameTheme` kısayolu.
extension GameThemeContext on BuildContext {
  GameTheme get gameTheme => Theme.of(this).extension<GameTheme>()!;
}
