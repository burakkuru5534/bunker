import 'package:flutter/painting.dart';

/// Renk token'ları (bkz. docs/ui/brand.md §3).
///
/// Widget'lar bu sabitleri doğrudan değil, `Theme.of(context)` ve
/// `GameTheme` üzerinden kullanır; sabitler yalnızca tema kurulumunda okunur.
abstract final class AppColors {
  // Yüzeyler
  static const background = Color(0xFF0F1115);

  /// Logonun zemini; yerel açılış ekranı, uygulama simgesi zemini
  /// (flutter_native_splash.yaml, flutter_launcher_icons.yaml) ve uygulama
  /// içi açılış ekranı aynı rengi kullanır — geçişte renk sıçraması olmaz.
  static const brandBlack = Color(0xFF030202);
  static const surface = Color(0xFF1A1D23);
  static const surfaceHigh = Color(0xFF242832);
  static const outline = Color(0xFF2E333D);

  // Metin
  static const textPrimary = Color(0xFFECEDEE);
  static const textSecondary = Color(0xFF9BA1A6);

  // Vurgular
  static const primary = Color(0xFFF2A33A);
  static const onPrimary = Color(0xFF1A1200);
  static const secondary = Color(0xFF4FB3A9);
  static const onSecondary = Color(0xFF0F1115);
  static const danger = Color(0xFFF66A6E);
  static const onDanger = Color(0xFF1A0000);
  static const success = Color(0xFF4CC38A);

  // Kart kategorileri
  static const categoryProfession = Color(0xFF6E9BF5);
  static const categoryHealth = Color(0xFFF27A93);
  static const categoryHobby = Color(0xFFA98BF5);
  static const categoryItem = Color(0xFF7CC46A);
  static const categorySecret = Color(0xFFE0C15A);

  /// Oyuncu avatarları için 8 renklik döngü (üzerindeki metin `background`).
  static const avatarPalette = <Color>[
    Color(0xFF6E9BF5),
    Color(0xFFF27A93),
    Color(0xFFA98BF5),
    Color(0xFF7CC46A),
    Color(0xFFE0C15A),
    Color(0xFF4FB3A9),
    Color(0xFFF2A33A),
    Color(0xFFF66A6E),
  ];
}
