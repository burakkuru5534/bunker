import 'package:flutter/material.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/theme/game_theme.dart';

enum StatusBadgeKind { ready, waiting, host, spectator, eliminated, left, turn }

/// Küçük hap biçimli durum etiketi ("Hazır", "Oda sahibi", "Elendi"…).
class StatusBadge extends StatelessWidget {
  const StatusBadge(this.kind, {super.key});

  final StatusBadgeKind kind;

  static String labelOf(AppLocalizations l, StatusBadgeKind kind) =>
      switch (kind) {
        StatusBadgeKind.ready => l.statusReady,
        StatusBadgeKind.waiting => l.statusWaiting,
        StatusBadgeKind.host => l.statusHost,
        StatusBadgeKind.spectator => l.statusSpectator,
        StatusBadgeKind.eliminated => l.statusEliminated,
        StatusBadgeKind.left => l.statusLeft,
        StatusBadgeKind.turn => l.statusTurn,
      };

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final game = context.gameTheme;
    final (Color fg, IconData? icon) = switch (kind) {
      StatusBadgeKind.ready => (game.hope, Icons.check_rounded),
      StatusBadgeKind.waiting => (scheme.onSurfaceVariant, null),
      StatusBadgeKind.host => (scheme.primary, Icons.star_rounded),
      StatusBadgeKind.spectator => (
        scheme.onSurfaceVariant,
        Icons.visibility_rounded,
      ),
      StatusBadgeKind.eliminated => (game.danger, null),
      StatusBadgeKind.left => (scheme.onSurfaceVariant, null),
      StatusBadgeKind.turn => (scheme.primary, null),
    };

    // Yükseklik en az 28; büyük yazı ölçeğinde büyür, dar alanda metin
    // kısaltılır (PLAY-012).
    return Container(
      constraints: const BoxConstraints(minHeight: 28),
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 2),
      decoration: BoxDecoration(
        color: scheme.surfaceContainerHigh,
        borderRadius: BorderRadius.circular(AppRadius.full),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          if (icon != null) ...[
            Icon(icon, size: 14, color: fg),
            const SizedBox(width: AppSpacing.s1),
          ],
          Flexible(
            child: Text(
              labelOf(context.l10n, kind),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.labelMedium
                  ?.copyWith(color: fg),
            ),
          ),
        ],
      ),
    );
  }
}
