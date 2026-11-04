import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/theme/game_theme.dart';
import 'package:siginak/core/ui/app_buttons.dart';
import 'package:siginak/core/ui/player_avatar.dart';
import 'package:siginak/features/game/data/game_repository.dart';
import 'package:siginak/features/game/domain/game_result.dart';
import 'package:siginak/features/game/presentation/elimination_panel.dart';
import 'package:siginak/features/game/presentation/game_widgets.dart';

/// Maç sonucu; `game.finished` kaçırıldıysa (ör. yeniden bağlanma) sunucudan.
/// Otomatik yeniden deneme yok: hata ekranında "Tekrar dene" var.
final gameResultProvider = FutureProvider.autoDispose
    .family<GameResult, String>(
      (ref, gameId) => ref.read(gameRepositoryProvider).fetchResult(gameId),
      retry: (_, _) => null,
    );

/// Maç sonu görünümü (docs/ui/screen-map.md §3.4d).
class ResultView extends StatefulWidget {
  const ResultView({
    required this.result,
    required this.myUserId,
    required this.isHost,
    required this.onRematch,
    required this.onGoHome,
    super.key,
  });

  final GameResult result;
  final String? myUserId;
  final bool isHost;
  final Future<void> Function() onRematch;
  final Future<void> Function() onGoHome;

  @override
  State<ResultView> createState() => _ResultViewState();
}

class _ResultViewState extends State<ResultView> {
  String? _busy;

  Future<void> _guard(String key, Future<void> Function() action) async {
    if (_busy != null) return;
    setState(() => _busy = key);
    try {
      await action();
    } finally {
      if (mounted) setState(() => _busy = null);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final r = widget.result;
    final game = context.gameTheme;

    Widget section(String title, Color color) => Padding(
      padding: const EdgeInsets.only(top: AppSpacing.s5, bottom: AppSpacing.s2),
      child: Text(
        title,
        style: theme.textTheme.titleMedium?.copyWith(color: color),
      ),
    );

    return Column(
      children: [
        Expanded(
          child: ListView(
            padding: const EdgeInsets.all(AppSpacing.screen),
            children: [
              Text(
                r.scenarioTitle,
                textAlign: TextAlign.center,
                style: theme.textTheme.titleSmall,
              ),
              if (r.endReason == 'abandoned')
                Padding(
                  padding: const EdgeInsets.only(top: AppSpacing.s2),
                  child: Text(
                    l.resultAbandoned,
                    textAlign: TextAlign.center,
                    style: theme.textTheme.bodyMedium,
                  ),
                ),
              section(l.resultWinners, game.hope),
              for (final (i, p) in r.players.indexed)
                if (p.isWinner)
                  _PlayerResult(
                    player: p,
                    seat: i,
                    isMe: p.userId == widget.myUserId,
                  ),
              section(l.resultLosers, game.danger),
              for (final (i, p) in r.players.indexed)
                if (!p.isWinner)
                  _PlayerResult(
                    player: p,
                    seat: i,
                    isMe: p.userId == widget.myUserId,
                  ),
              if (r.rounds.isNotEmpty) ...[
                section(l.resultRounds, theme.colorScheme.onSurface),
                for (final round in r.rounds)
                  Padding(
                    padding: const EdgeInsets.only(bottom: AppSpacing.s1),
                    child: Text(
                      l.resultRoundLine(
                        round.round,
                        r.nicknameOf(round.eliminatedUserId),
                        _roundMethod(l, round),
                      ),
                      style: theme.textTheme.bodyMedium,
                    ),
                  ),
              ],
            ],
          ),
        ),
        Material(
          color: theme.colorScheme.surfaceContainer,
          child: Padding(
            padding: const EdgeInsets.all(AppSpacing.screen),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                if (widget.isHost) ...[
                  AppButton.primary(
                    label: l.resultRematch,
                    icon: Icons.replay_rounded,
                    isLoading: _busy == 'rematch',
                    onPressed: _busy == null
                        ? () => unawaited(_guard('rematch', widget.onRematch))
                        : null,
                  ),
                  const SizedBox(height: AppSpacing.s3),
                ] else
                  Padding(
                    padding: const EdgeInsets.only(bottom: AppSpacing.s3),
                    child: Text(
                      l.resultWaitingHost,
                      textAlign: TextAlign.center,
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ),
                AppButton.secondary(
                  label: l.actionGoHome,
                  icon: Icons.home_outlined,
                  isLoading: _busy == 'home',
                  onPressed: _busy == null
                      ? () => unawaited(_guard('home', widget.onGoHome))
                      : null,
                ),
              ],
            ),
          ),
        ),
      ],
    );
  }

  static String _roundMethod(AppLocalizations l, ResultRound r) {
    final method = methodShort(l, r.method);
    if (r.method != 'vote') return method;
    final votes = r.tally
        .where((t) => t.userId == r.eliminatedUserId)
        .fold(0, (s, t) => s + t.votes);
    return '$method, ${l.gameVotesCount(votes)}';
  }
}

class _PlayerResult extends StatelessWidget {
  const _PlayerResult({
    required this.player,
    required this.seat,
    required this.isMe,
  });

  final ResultPlayer player;
  final int seat;
  final bool isMe;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final p = player;
    return Card(
      key: ValueKey('result-${p.userId}'),
      child: ExpansionTile(
        leading: PlayerAvatar(
          nickname: p.nickname,
          seat: seat,
          size: PlayerAvatarSize.small,
          presence: p.isWinner
              ? PlayerPresence.online
              : PlayerPresence.eliminated,
        ),
        title: Text(isMe ? '${p.nickname} (${l.roomYou})' : p.nickname),
        subtitle: p.cards.isEmpty ? null : Text(p.cards.first.text),
        childrenPadding: const EdgeInsets.fromLTRB(
          AppSpacing.s4,
          0,
          AppSpacing.s4,
          AppSpacing.s3,
        ),
        expandedCrossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (final c in p.cards) CardChip(category: c.category, text: c.text),
        ],
      ),
    );
  }
}
