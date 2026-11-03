import 'package:flutter/animation.dart';

/// Boşluk token'ları — 4'ün katları (docs/ui/design-system.md §1).
abstract final class AppSpacing {
  static const double s1 = 4;
  static const double s2 = 8;
  static const double s3 = 12;
  static const double s4 = 16;
  static const double s5 = 24;
  static const double s6 = 32;
  static const double s8 = 48;

  /// Ekran kenar boşluğu.
  static const double screen = s4;

  /// En küçük dokunma alanı (erişilebilirlik).
  static const double minTouchTarget = 48;
}

/// Köşe yuvarlaklığı token'ları.
abstract final class AppRadius {
  static const double s = 8;
  static const double m = 12;
  static const double l = 16;
  static const double xl = 24;
  static const double full = 999;
}

/// Hareket süreleri ve eğrileri.
abstract final class AppMotion {
  static const fast = Duration(milliseconds: 150);
  static const medium = Duration(milliseconds: 300);
  static const slow = Duration(milliseconds: 500);

  static const Curve enter = Curves.easeOutCubic;
  static const Curve exit = Curves.easeInCubic;
}
