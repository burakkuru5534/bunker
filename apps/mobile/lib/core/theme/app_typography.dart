import 'package:flutter/material.dart';

/// Yazı tipi aileleri (assets/fonts, pubspec.yaml).
abstract final class AppFonts {
  /// Başlık / gösterim: 500, 600, 700.
  static const display = 'Oswald';

  /// Gövde / arayüz: 400, 500, 600.
  static const body = 'Inter';
}

/// Uygulamanın metin stilleri (docs/ui/brand.md §4).
TextTheme buildTextTheme(Color primary, Color secondary) {
  TextStyle oswald(double size, double height, FontWeight w) => TextStyle(
    fontFamily: AppFonts.display,
    fontSize: size,
    height: height / size,
    fontWeight: w,
    color: primary,
  );
  TextStyle inter(double size, double height, FontWeight w, [Color? c]) =>
      TextStyle(
        fontFamily: AppFonts.body,
        fontSize: size,
        height: height / size,
        fontWeight: w,
        color: c ?? primary,
      );

  return TextTheme(
    displayLarge: oswald(40, 48, FontWeight.w700),
    displayMedium: oswald(34, 40, FontWeight.w700),
    displaySmall: oswald(28, 36, FontWeight.w700),
    headlineLarge: oswald(32, 40, FontWeight.w600),
    headlineMedium: oswald(28, 36, FontWeight.w600),
    headlineSmall: oswald(24, 32, FontWeight.w600),
    titleLarge: oswald(22, 28, FontWeight.w500),
    titleMedium: oswald(18, 24, FontWeight.w500),
    titleSmall: inter(14, 20, FontWeight.w600),
    bodyLarge: inter(16, 24, FontWeight.w400),
    bodyMedium: inter(14, 20, FontWeight.w400),
    bodySmall: inter(12, 16, FontWeight.w400, secondary),
    labelLarge: inter(14, 20, FontWeight.w600),
    labelMedium: inter(12, 16, FontWeight.w600),
    labelSmall: inter(11, 16, FontWeight.w500, secondary),
  );
}

/// Türkçe kurallarıyla büyük harfe çevirir (`i → İ`, `ı → I`).
///
/// Dart'ın `toUpperCase()` yerel ayardan bağımsızdır ve `i`'yi `I` yapar.
String turkishUpperCase(String input) =>
    input.replaceAll('i', 'İ').replaceAll('ı', 'I').toUpperCase();

/// Türkçe kurallarıyla küçük harfe çevirir (`I → ı`, `İ → i`).
String turkishLowerCase(String input) =>
    input.replaceAll('I', 'ı').replaceAll('İ', 'i').toLowerCase();
