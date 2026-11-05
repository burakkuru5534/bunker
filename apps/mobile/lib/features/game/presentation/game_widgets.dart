import 'dart:async';

import 'package:flutter/material.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/theme/game_theme.dart';
import 'package:siginak/core/ui/player_avatar.dart';
import 'package:siginak/core/ui/status_badge.dart';
import 'package:siginak/features/game/domain/game_models.dart';

/// Kategorinin renk / ikon / etiketi.
CategoryStyle categoryStyle(BuildContext context, CardCategory c) {
  final g = context.gameTheme;
  return switch (c) {
    CardCategory.profession => g.profession,
    CardCategory.health => g.health,
    CardCategory.hobby => g.hobby,
    CardCategory.item => g.item,
    CardCategory.secret => g.secret,
  };
}

String phaseLabel(AppLocalizations l, GamePhase p) => switch (p) {
  GamePhase.dealing => l.gamePhaseDealing,
  GamePhase.scenarioIntro => l.gamePhaseScenario,
  GamePhase.reveal => l.gamePhaseReveal,
  GamePhase.discussion => l.gamePhaseDiscussion,
  GamePhase.voting => l.gamePhaseVoting,
  GamePhase.tieBreak => l.gamePhaseTieBreak,
  GamePhase.elimination => l.gamePhaseElimination,
  GamePhase.finished => l.gamePhaseFinished,
};

/// "01:12" biçimi.
String formatCountdown(Duration d) {
  final s = d.inSeconds;
  return '${(s ~/ 60).toString().padLeft(2, '0')}:'
      '${(s % 60).toString().padLeft(2, '0')}';
}

/// Sunucu saatine göre geri sayım; saniyede bir yenilenir.
class PhaseCountdown extends StatefulWidget {
  const PhaseCountdown({
    required this.game,
    this.style,
    this.clock = DateTime.now,
    super.key,
  });

  final GameView game;
  final TextStyle? style;

  /// Cihaz saati (testlerde sahte saat verilir).
  final DateTime Function() clock;

  @override
  State<PhaseCountdown> createState() => _PhaseCountdownState();
}

class _PhaseCountdownState extends State<PhaseCountdown> {
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    _timer = Timer.periodic(const Duration(seconds: 1), (_) => setState(() {}));
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final left = widget.game.remaining(widget.clock());
    if (left == null) return const SizedBox.shrink();
    // Yukarı yuvarla: 0,4 sn kala "00:01" (sıfır, süre gerçekten bitince).
    final shown = Duration(seconds: (left.inMilliseconds / 1000).ceil());
    final text = formatCountdown(shown);
    return Semantics(
      label: context.l10n.gameCountdownLabel(text),
      excludeSemantics: true,
      child: Text(
        text,
        key: const ValueKey('phase-countdown'),
        style: widget.style,
      ),
    );
  }
}

/// Üst çubuk: senaryo düğmesi, tur, aşama ve kalan süre.
class PhaseBar extends StatelessWidget {
  const PhaseBar({
    required this.game,
    required this.onShowScenario,
    this.clock = DateTime.now,
    super.key,
  });

  final GameView game;
  final VoidCallback onShowScenario;
  final DateTime Function() clock;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    return Material(
      color: theme.colorScheme.surfaceContainer,
      child: Padding(
        padding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.s3,
          vertical: AppSpacing.s2,
        ),
        child: Row(
          children: [
            Expanded(
              child: Align(
                alignment: AlignmentDirectional.centerStart,
                child: game.scenario == null
                    ? const SizedBox.shrink()
                    : ActionChip(
                        avatar: const Icon(Icons.public_rounded, size: 18),
                        label: Text(
                          game.scenario!.title,
                          overflow: TextOverflow.ellipsis,
                        ),
                        tooltip: l.gameShowScenario,
                        onPressed: onShowScenario,
                      ),
              ),
            ),
            const SizedBox(width: AppSpacing.s2),
            Column(
              crossAxisAlignment: CrossAxisAlignment.end,
              children: [
                if (game.round > 0)
                  Text(
                    l.gameRound(game.round, game.totalRounds),
                    style: theme.textTheme.labelMedium,
                  ),
                Text(
                  phaseLabel(l, game.phase).toUpperCase(),
                  style: theme.textTheme.labelLarge?.copyWith(
                    color: theme.colorScheme.primary,
                  ),
                ),
              ],
            ),
            const SizedBox(width: AppSpacing.s3),
            PhaseCountdown(
              game: game,
              clock: clock,
              style: theme.textTheme.titleLarge,
            ),
          ],
        ),
      ),
    );
  }
}

/// Senaryo içeriği (tanıtım görünümü ve senaryo penceresi).
class ScenarioDetails extends StatelessWidget {
  const ScenarioDetails({
    required this.scenario,
    required this.capacity,
    super.key,
  });

  final ScenarioInfo scenario;
  final int capacity;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    Widget heading(String text) => Padding(
      padding: const EdgeInsets.only(top: AppSpacing.s4, bottom: AppSpacing.s1),
      child: Text(text, style: theme.textTheme.titleSmall),
    );
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(scenario.title, style: theme.textTheme.headlineSmall),
        const SizedBox(height: AppSpacing.s1),
        Text(
          l.gameScenarioDuration(scenario.duration),
          style: theme.textTheme.labelLarge?.copyWith(
            color: theme.colorScheme.primary,
          ),
        ),
        heading(l.gameScenarioDisaster),
        Text(scenario.disaster, style: theme.textTheme.bodyMedium),
        heading(l.gameScenarioBunker),
        for (final f in scenario.bunker)
          Padding(
            padding: const EdgeInsets.only(bottom: AppSpacing.s1),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text('•  '),
                Expanded(child: Text(f, style: theme.textTheme.bodyMedium)),
              ],
            ),
          ),
        heading(l.gameScenarioOutside),
        Text(scenario.outside, style: theme.textTheme.bodyMedium),
        const SizedBox(height: AppSpacing.s4),
        Text(
          l.gameScenarioCapacity(capacity),
          style: theme.textTheme.titleSmall?.copyWith(
            color: context.gameTheme.hope,
          ),
        ),
      ],
    );
  }
}

/// Açılmış kart çipi.
class CardChip extends StatelessWidget {
  const CardChip({required this.category, required this.text, super.key});

  final CardCategory category;
  final String text;

  @override
  Widget build(BuildContext context) {
    final style = categoryStyle(context, category);
    final theme = Theme.of(context);
    return Semantics(
      label: '${style.label}: $text',
      excludeSemantics: true,
      child: Container(
        margin: const EdgeInsets.only(top: AppSpacing.s1),
        padding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.s2,
          vertical: 2,
        ),
        decoration: BoxDecoration(
          color: style.color.withValues(alpha: 0.16),
          borderRadius: BorderRadius.circular(AppRadius.s),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(style.icon, size: 14, color: style.color),
            const SizedBox(width: AppSpacing.s1),
            Flexible(
              child: Text(
                text,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodySmall,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Oyuncu kutusu: avatar, ad, durum ve açılmış kartlar.
class PlayerTile extends StatelessWidget {
  const PlayerTile({
    required this.participant,
    required this.isMe,
    required this.isTurn,
    this.onVote,
    this.onMenu,
    super.key,
  });

  final Participant participant;
  final bool isMe;
  final bool isTurn;

  /// Doluysa oyuncu oylanabilir: kutu vurgulanır, dokununca çağrılır.
  final VoidCallback? onVote;

  /// Oylama dışında dokununca: oyuncu menüsü (Raporla / Engelle).
  final VoidCallback? onMenu;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final p = participant;
    final presence = !p.isAlive
        ? PlayerPresence.eliminated
        : p.connected
        ? PlayerPresence.online
        : PlayerPresence.disconnected;
    final badge = switch (p.status) {
      ParticipantStatus.eliminated => StatusBadgeKind.eliminated,
      ParticipantStatus.left => StatusBadgeKind.left,
      ParticipantStatus.alive => isTurn ? StatusBadgeKind.turn : null,
    };
    final highlight = isTurn
        ? theme.colorScheme.primary
        : onVote != null
        ? context.gameTheme.danger
        : null;
    return Card(
      key: ValueKey('player-${p.userId}'),
      clipBehavior: Clip.antiAlias,
      shape: highlight != null
          ? RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(AppRadius.l),
              side: BorderSide(color: highlight, width: 2),
            )
          : null,
      child: InkWell(
        onTap: onVote ?? onMenu,
        child: Semantics(
          button: onVote != null,
          hint: onVote != null ? l.gameVoteFor(p.nickname) : null,
          child: Padding(
            padding: const EdgeInsets.all(AppSpacing.s3),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    PlayerAvatar(
                      nickname: p.nickname,
                      seat: p.seat,
                      size: PlayerAvatarSize.small,
                      presence: presence,
                    ),
                    const SizedBox(width: AppSpacing.s2),
                    Expanded(
                      child: Text(
                        isMe ? '${p.nickname} (${l.roomYou})' : p.nickname,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: theme.textTheme.titleSmall,
                      ),
                    ),
                  ],
                ),
                if (badge != null) ...[
                  const SizedBox(height: AppSpacing.s1),
                  StatusBadge(badge),
                ],
                for (final c in CardCategory.values)
                  if (p.revealed[c] case final text?)
                    CardChip(category: c, text: text),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

/// "Kartlarım" alt sayfası: kendi 5 kartım, açık/kapalı işaretli.
Future<void> showMyCardsSheet(BuildContext context, List<MyCard> cards) =>
    showModalBottomSheet<void>(
      context: context,
      showDragHandle: true,
      isScrollControlled: true,
      builder: (context) => _MyCardsSheet(cards: cards),
    );

class _MyCardsSheet extends StatelessWidget {
  const _MyCardsSheet({required this.cards});

  final List<MyCard> cards;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(
          AppSpacing.screen,
          0,
          AppSpacing.screen,
          AppSpacing.screen,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(l.gameMyCards, style: theme.textTheme.titleLarge),
            const SizedBox(height: AppSpacing.s3),
            if (cards.isEmpty) Text(l.gameNoCards),
            for (final c in cards)
              Builder(
                builder: (context) {
                  final style = categoryStyle(context, c.category);
                  return ListTile(
                    key: ValueKey('my-card-${c.category.name}'),
                    contentPadding: EdgeInsets.zero,
                    leading: Icon(style.icon, color: style.color),
                    title: Text(c.text),
                    subtitle: Text(style.label),
                    trailing: Chip(
                      label: Text(
                        c.revealed ? l.gameCardOpen : l.gameCardClosed,
                      ),
                      avatar: Icon(
                        c.revealed
                            ? Icons.visibility_rounded
                            : Icons.visibility_off_rounded,
                        size: 16,
                      ),
                    ),
                  );
                },
              ),
          ],
        ),
      ),
    );
  }
}
