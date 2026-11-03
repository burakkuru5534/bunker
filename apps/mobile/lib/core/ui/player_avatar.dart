import 'package:flutter/material.dart';

import 'package:siginak/core/theme/app_typography.dart';
import 'package:siginak/core/theme/game_theme.dart';

enum PlayerAvatarSize {
  small(28),
  medium(40),
  large(64);

  const PlayerAvatarSize(this.dimension);
  final double dimension;
}

enum PlayerPresence { online, disconnected, eliminated }

/// Baş harf + koltuk sırasına göre sabit renkli oyuncu avatarı.
class PlayerAvatar extends StatelessWidget {
  const PlayerAvatar({
    required this.nickname,
    required this.seat,
    this.size = PlayerAvatarSize.medium,
    this.presence = PlayerPresence.online,
    super.key,
  });

  final String nickname;

  /// Odadaki sırası (0'dan başlar); rengi belirler.
  final int seat;
  final PlayerAvatarSize size;
  final PlayerPresence presence;

  /// Takma adın ilk harfi, Türkçe kurallarıyla büyük harf.
  static String initialOf(String nickname) {
    final trimmed = nickname.trim();
    if (trimmed.isEmpty) return '?';
    return turkishUpperCase(String.fromCharCode(trimmed.runes.first));
  }

  @override
  Widget build(BuildContext context) {
    final game = context.gameTheme;
    final scheme = Theme.of(context).colorScheme;
    final d = size.dimension;
    final color = game.avatarColorFor(seat);
    final faded = presence != PlayerPresence.online;

    final stateLabel = switch (presence) {
      PlayerPresence.online => '',
      PlayerPresence.disconnected => ', bağlantısı koptu',
      PlayerPresence.eliminated => ', elendi',
    };

    return Semantics(
      label: '$nickname$stateLabel',
      image: true,
      child: SizedBox.square(
        dimension: d,
        child: Stack(
          clipBehavior: Clip.none,
          children: [
            Opacity(
              opacity: faded ? 0.45 : 1,
              child: Container(
                alignment: Alignment.center,
                decoration: BoxDecoration(color: color, shape: BoxShape.circle),
                child: ExcludeSemantics(
                  child: Text(
                    initialOf(nickname),
                    style: TextStyle(
                      fontFamily: AppFonts.body,
                      fontWeight: FontWeight.w600,
                      fontSize: d * 0.42,
                      height: 1,
                      color: game.onAvatar,
                    ),
                  ),
                ),
              ),
            ),
            if (presence == PlayerPresence.disconnected)
              Positioned(
                right: -1,
                bottom: -1,
                child: Container(
                  key: const ValueKey('avatar-disconnected-dot'),
                  width: d * 0.3,
                  height: d * 0.3,
                  decoration: BoxDecoration(
                    color: scheme.onSurfaceVariant,
                    shape: BoxShape.circle,
                    border: Border.all(color: scheme.surface, width: 2),
                  ),
                ),
              ),
            if (presence == PlayerPresence.eliminated)
              Positioned.fill(
                child: CustomPaint(
                  key: const ValueKey('avatar-eliminated-strike'),
                  painter: _StrikePainter(game.danger),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _StrikePainter extends CustomPainter {
  _StrikePainter(this.color);

  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..strokeWidth = 2
      ..strokeCap = StrokeCap.round;
    canvas.drawLine(
      Offset(size.width * 0.15, size.height * 0.85),
      Offset(size.width * 0.85, size.height * 0.15),
      paint,
    );
  }

  @override
  bool shouldRepaint(_StrikePainter old) => old.color != color;
}
