import 'package:flutter/material.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/theme/game_theme.dart';
import 'package:siginak/core/ui/player_avatar.dart';
import 'package:siginak/features/game/domain/game_models.dart';
import 'package:siginak/features/game/presentation/game_widgets.dart';

/// Eleme yöntemi → uzun metin.
String methodLabel(AppLocalizations l, String method) => switch (method) {
  'spectators' => l.gameMethodSpectators,
  'lottery' => l.gameMethodLottery,
  'left' => l.gameMethodLeft,
  _ => l.gameMethodVote,
};

/// Eleme yöntemi → kısa metin (tur özeti).
String methodShort(AppLocalizations l, String method) => switch (method) {
  'spectators' => l.gameMethodShortSpectators,
  'lottery' => l.gameMethodShortLottery,
  'left' => l.gameMethodShortLeft,
  _ => l.gameMethodShortVote,
};

/// Eleme sonucu (docs/ui/screen-map.md §3.4c "ELIMINATION"): elenen, yöntem,
/// adaylara gelen oy sayıları (kimin kime verdiği yok), elenenin kartları.
class EliminationPanel extends StatelessWidget {
  const EliminationPanel({required this.game, required this.result, super.key});

  final GameView game;
  final Elimination result;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final out = game.participantOf(result.userId);
    String nameOf(String id) => game.participantOf(id)?.nickname ?? '';
    final tally = [...result.tally]..sort((a, b) => b.votes.compareTo(a.votes));

    return SingleChildScrollView(
      key: const ValueKey('elimination-panel'),
      padding: const EdgeInsets.all(AppSpacing.screen),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          if (out != null)
            Center(
              child: PlayerAvatar(
                nickname: out.nickname,
                seat: out.seat,
                size: PlayerAvatarSize.large,
                presence: PlayerPresence.eliminated,
              ),
            ),
          const SizedBox(height: AppSpacing.s3),
          Text(
            l.gameEliminated(out?.nickname ?? ''),
            textAlign: TextAlign.center,
            style: theme.textTheme.headlineSmall,
          ),
          const SizedBox(height: AppSpacing.s1),
          Text(
            methodLabel(l, result.method),
            textAlign: TextAlign.center,
            style: theme.textTheme.titleSmall?.copyWith(
              color: context.gameTheme.danger,
            ),
          ),
          const SizedBox(height: AppSpacing.s4),
          if (result.method == 'vote' || result.method == 'lottery') ...[
            if (tally.isEmpty)
              Text(l.gameNoVotes, textAlign: TextAlign.center)
            else
              for (final t in tally)
                ListTile(
                  dense: true,
                  title: Text(nameOf(t.userId)),
                  trailing: Text(
                    l.gameVotesCount(t.votes),
                    style: theme.textTheme.titleSmall,
                  ),
                ),
          ],
          if (result.lotteryCandidates.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: AppSpacing.s2),
              child: Text(
                l.gameLotteryAmong(
                  result.lotteryCandidates.map(nameOf).join(', '),
                ),
                textAlign: TextAlign.center,
                style: theme.textTheme.bodyMedium,
              ),
            ),
          if (out != null && out.revealed.isNotEmpty) ...[
            const SizedBox(height: AppSpacing.s4),
            Text(l.gameEliminatedCards, style: theme.textTheme.titleSmall),
            for (final c in CardCategory.values)
              if (out.revealed[c] case final text?)
                CardChip(category: c, text: text),
          ],
        ],
      ),
    );
  }
}

/// "Kapı kapandı. Artık izleyicisin." — elendiğim an kısa süre gösterilir.
class DoorClosedOverlay extends StatelessWidget {
  const DoorClosedOverlay({super.key});

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    return Positioned.fill(
      child: Semantics(
        liveRegion: true,
        child: ColoredBox(
          key: const ValueKey('door-closed'),
          color: theme.colorScheme.surface.withValues(alpha: 0.96),
          child: Center(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  Icons.door_front_door_rounded,
                  size: 72,
                  color: context.gameTheme.danger,
                ),
                const SizedBox(height: AppSpacing.s4),
                Text(
                  l.gameDoorClosedTitle,
                  style: theme.textTheme.headlineMedium,
                ),
                const SizedBox(height: AppSpacing.s2),
                Text(l.gameDoorClosedBody, style: theme.textTheme.titleMedium),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
