import 'dart:async';

import 'package:flutter/material.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/ui/state_views.dart';
import 'package:siginak/features/game/domain/game_models.dart';
import 'package:siginak/features/game/presentation/game_widgets.dart';

/// Oyun görünümü (docs/ui/screen-map.md §3.4b–c): üst çubuk, senaryo
/// tanıtımı veya oyuncu ızgarası + aşama bilgisi, "Kartlarım".
///
/// Yalnızca sunucu durumunu gösterir; eylemler (kart açma, hazır, oy)
/// PLAY-003/004/007'de eklenir.
class GameBody extends StatelessWidget {
  const GameBody({
    required this.game,
    required this.myUserId,
    this.clock = DateTime.now,
    super.key,
  });

  final GameView game;
  final String? myUserId;

  /// Cihaz saati (testlerde sahte saat verilir).
  final DateTime Function() clock;

  void _showScenario(BuildContext context) {
    final scenario = game.scenario;
    if (scenario == null) return;
    unawaited(
      showModalBottomSheet<void>(
        context: context,
        showDragHandle: true,
        isScrollControlled: true,
        builder: (context) => SafeArea(
          child: SingleChildScrollView(
            padding: const EdgeInsets.fromLTRB(
              AppSpacing.screen,
              0,
              AppSpacing.screen,
              AppSpacing.screen,
            ),
            child: ScenarioDetails(scenario: scenario, capacity: game.capacity),
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final me = game.participantOf(myUserId);
    final spectator = me == null || !me.isAlive;
    final intro =
        game.phase == GamePhase.dealing ||
        game.phase == GamePhase.scenarioIntro;

    return Column(
      children: [
        PhaseBar(
          game: game,
          clock: clock,
          onShowScenario: () => _showScenario(context),
        ),
        if (spectator && !intro) _SpectatorBanner(text: l.gameSpectatorBanner),
        Expanded(
          child: intro
              ? _IntroView(game: game)
              : _PlayersView(game: game, myUserId: myUserId),
        ),
        if (!intro) _InfoBar(game: game, myUserId: myUserId),
        _BottomBar(cards: game.myCards),
      ],
    );
  }
}

class _IntroView extends StatelessWidget {
  const _IntroView({required this.game});

  final GameView game;

  @override
  Widget build(BuildContext context) {
    final scenario = game.scenario;
    if (game.phase == GamePhase.dealing || scenario == null) {
      return LoadingView(message: context.l10n.gameDealingInfo);
    }
    return SingleChildScrollView(
      padding: const EdgeInsets.all(AppSpacing.screen),
      child: ScenarioDetails(scenario: scenario, capacity: game.capacity),
    );
  }
}

class _PlayersView extends StatelessWidget {
  const _PlayersView({required this.game, required this.myUserId});

  final GameView game;
  final String? myUserId;

  @override
  Widget build(BuildContext context) {
    return GridView.builder(
      padding: const EdgeInsets.all(AppSpacing.s3),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 2,
        mainAxisSpacing: AppSpacing.s2,
        crossAxisSpacing: AppSpacing.s2,
        mainAxisExtent: 210,
      ),
      itemCount: game.participants.length,
      itemBuilder: (context, i) {
        final p = game.participants[i];
        return PlayerTile(
          participant: p,
          isMe: p.userId == myUserId,
          isTurn: game.phase == GamePhase.reveal && p.userId == game.turnUserId,
        );
      },
    );
  }
}

/// Aşamaya göre bilgi satırı (eylemler sonraki görevlerde).
class _InfoBar extends StatelessWidget {
  const _InfoBar({required this.game, required this.myUserId});

  final GameView game;
  final String? myUserId;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    String nameOf(String? id) => game.participantOf(id)?.nickname ?? '';
    final (String title, String? detail) = switch (game.phase) {
      GamePhase.reveal => (
        game.turnUserId == myUserId
            ? l.gameRevealMine
            : l.gameRevealOther(nameOf(game.turnUserId)),
        null,
      ),
      GamePhase.discussion => (
        l.gameDiscussionInfo,
        l.gameReadyProgress(game.readyProgress.done, game.readyProgress.total),
      ),
      GamePhase.voting => (
        l.gameVotingInfo,
        l.gameVoteProgress(game.voteProgress.done, game.voteProgress.total),
      ),
      GamePhase.tieBreak => (l.gameTieBreakInfo, null),
      GamePhase.elimination => (
        game.lastElimination == null
            ? phaseLabel(l, game.phase)
            : l.gameEliminated(nameOf(game.lastElimination!.userId)),
        null,
      ),
      GamePhase.finished => (l.gamePhaseFinished, null),
      GamePhase.dealing || GamePhase.scenarioIntro => ('', null),
    };
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(AppSpacing.s4),
      color: theme.colorScheme.surfaceContainerHigh,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            title,
            key: const ValueKey('phase-info'),
            style: theme.textTheme.titleMedium,
          ),
          if (detail != null) ...[
            const SizedBox(height: AppSpacing.s1),
            Text(detail, style: theme.textTheme.bodyMedium),
          ],
        ],
      ),
    );
  }
}

class _SpectatorBanner extends StatelessWidget {
  const _SpectatorBanner({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(
        horizontal: AppSpacing.s4,
        vertical: AppSpacing.s2,
      ),
      color: theme.colorScheme.secondaryContainer,
      child: Text(
        text,
        style: theme.textTheme.labelLarge?.copyWith(
          color: theme.colorScheme.onSecondaryContainer,
        ),
      ),
    );
  }
}

class _BottomBar extends StatelessWidget {
  const _BottomBar({required this.cards});

  final List<MyCard> cards;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    return Padding(
      padding: const EdgeInsets.all(AppSpacing.s3),
      child: Row(
        children: [
          FilledButton.tonalIcon(
            onPressed: () => unawaited(showMyCardsSheet(context, cards)),
            icon: const Icon(Icons.style_rounded),
            label: Text(l.gameMyCards),
          ),
        ],
      ),
    );
  }
}
