import 'dart:math' as math;

import 'package:flutter/material.dart';

import 'package:siginak/features/game/presentation/intro/intro_timeline.dart';

/// Senaryoya özgü parçacıklar. Durum tutmaz: her parçacığın yeri yalnızca
/// sırasına ve `t`'ye bağlıdır (herhangi bir andan başlanabilir, testte
/// aynı kare aynı çizilir).
class IntroParticlesPainter extends CustomPainter {
  IntroParticlesPainter({
    required this.kind,
    required this.t,
    required this.intensity,
  });

  final ParticleKind kind;
  final double t;

  /// 0–1.
  final double intensity;

  static double _hash(int i, int salt) {
    final x = math.sin(i * 127.1 + salt * 311.7) * 43758.5453;
    return x - x.floorToDouble();
  }

  @override
  void paint(Canvas canvas, Size size) {
    if (intensity <= 0 || size.isEmpty) return;
    switch (kind) {
      case ParticleKind.ash:
        _ash(canvas, size);
      case ParticleKind.fog:
        _fog(canvas, size);
      case ParticleKind.embers:
        _embers(canvas, size);
      case ParticleKind.scan:
        _scan(canvas, size);
      case ParticleKind.sparks:
        _sparks(canvas, size);
    }
  }

  void _ash(Canvas canvas, Size size) {
    final n = (90 * intensity).round();
    final paint = Paint();
    for (var i = 0; i < n; i++) {
      final speed = 0.03 + 0.05 * _hash(i, 1);
      final y = (_hash(i, 2) + t * speed) % 1.0;
      final x = (_hash(i, 3) + 0.02 * math.sin(t * 0.8 + i)) % 1.0;
      final r = 1.0 + 2.2 * _hash(i, 4);
      paint.color = Color.lerp(
        const Color(0xFFBFC6CC),
        Colors.white,
        _hash(i, 5),
      )!.withValues(alpha: 0.35 + 0.45 * _hash(i, 6));
      canvas.drawCircle(Offset(x * size.width, y * size.height), r, paint);
    }
  }

  void _fog(Canvas canvas, Size size) {
    final paint = Paint()
      ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 60);
    for (var i = 0; i < 9; i++) {
      final speed = 0.008 + 0.012 * _hash(i, 1);
      final x = (_hash(i, 2) + t * speed) % 1.3 - 0.15;
      final y = 0.25 + 0.65 * _hash(i, 3);
      final r = size.width * (0.25 + 0.2 * _hash(i, 4));
      paint.color = const Color(0xFFB3261E)
          .withValues(alpha: (0.10 + 0.08 * _hash(i, 5)) * intensity);
      canvas.drawCircle(Offset(x * size.width, y * size.height), r, paint);
    }
  }

  void _embers(Canvas canvas, Size size) {
    final n = (70 * intensity).round();
    final paint = Paint()
      ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 1.5);
    for (var i = 0; i < n; i++) {
      final speed = 0.04 + 0.08 * _hash(i, 1);
      final y = 1 - (_hash(i, 2) + t * speed) % 1.0;
      final x = (_hash(i, 3) + 0.03 * math.sin(t * 1.7 + i * 2)) % 1.0;
      final flick = 0.5 + 0.5 * math.sin(t * (6 + 6 * _hash(i, 4)) + i);
      paint.color = Color.lerp(
        const Color(0xFFFF7A1A),
        const Color(0xFFFFD27A),
        _hash(i, 5),
      )!.withValues(alpha: 0.3 + 0.6 * flick);
      canvas.drawCircle(
        Offset(x * size.width, y * size.height),
        1.2 + 2 * _hash(i, 6),
        paint,
      );
    }
  }

  void _scan(Canvas canvas, Size size) {
    // Aşağı inen tarama çizgisi (4 sn'de bir).
    final y = (t / 4 % 1.0) * size.height;
    final band = Rect.fromLTWH(0, y - 40, size.width, 80);
    canvas.drawRect(
      band,
      Paint()
        ..shader = LinearGradient(
          begin: Alignment.topCenter,
          end: Alignment.bottomCenter,
          colors: [
            Colors.transparent,
            const Color(0xFFFF2D2D).withValues(alpha: 0.22 * intensity),
            Colors.transparent,
          ],
        ).createShader(band),
    );
    canvas.drawLine(
      Offset(0, y),
      Offset(size.width, y),
      Paint()
        ..color = const Color(0xFFFF4A4A).withValues(alpha: 0.6 * intensity)
        ..strokeWidth = 1.2,
    );
    // Uzakta yanıp sönen dron ışıkları.
    final dot = Paint();
    for (var i = 0; i < 7; i++) {
      final on = math.sin(t * (2 + 2 * _hash(i, 1)) + i * 1.7) > 0.3;
      if (!on) continue;
      dot.color = const Color(0xFFFF3B3B).withValues(alpha: 0.8 * intensity);
      canvas.drawCircle(
        Offset(
          ((_hash(i, 2) + t * 0.01 * (_hash(i, 3) - 0.5)) % 1.0) * size.width,
          (0.08 + 0.35 * _hash(i, 4)) * size.height,
        ),
        1.8,
        dot,
      );
    }
  }

  void _sparks(Canvas canvas, Size size) {
    final n = (60 * intensity).round();
    final paint = Paint()
      ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 2);
    for (var i = 0; i < n; i++) {
      final tw = math.sin(t * (3 + 5 * _hash(i, 1)) + i * 3.3);
      if (tw < 0) continue;
      paint.color = Color.lerp(
        const Color(0xFF7CFFB2),
        const Color(0xFFC59BFF),
        _hash(i, 2),
      )!.withValues(alpha: 0.7 * tw);
      canvas.drawCircle(
        Offset(_hash(i, 3) * size.width, _hash(i, 4) * 0.7 * size.height),
        1 + 2.5 * _hash(i, 5),
        paint,
      );
    }
    // Ara sıra elektrik çakması.
    final strike = (t * 0.7) % 1.0;
    if (strike < 0.05 && intensity > 0.6) {
      final x = _hash((t * 0.7).floor(), 9) * size.width;
      final path = Path()..moveTo(x, 0);
      var px = x;
      for (var k = 1; k <= 6; k++) {
        px += (_hash(k, (t * 0.7).floor()) - 0.5) * 40;
        path.lineTo(px, size.height * 0.08 * k);
      }
      canvas.drawPath(
        path,
        Paint()
          ..style = PaintingStyle.stroke
          ..strokeWidth = 1.5
          ..color = Colors.white.withValues(alpha: 0.7),
      );
    }
  }

  @override
  bool shouldRepaint(IntroParticlesPainter old) =>
      old.t != t || old.intensity != intensity || old.kind != kind;
}
