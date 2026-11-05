import 'dart:async';

import 'package:flutter/material.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/ui/app_buttons.dart';
import 'package:siginak/core/ui/confirm_dialog.dart';
import 'package:siginak/core/ui/state_views.dart';
import 'package:siginak/features/game/data/game_repository.dart';
import 'package:siginak/features/game/domain/game_models.dart';
import 'package:siginak/features/game/presentation/elimination_panel.dart';
import 'package:siginak/features/game/presentation/game_widgets.dart';

/// Oyun görünümü (docs/ui/screen-map.md §3.4b–c): üst çubuk, senaryo
/// tanıtımı veya oyuncu ızgarası + eylem alanı, "Kartlarım".
///
/// Sunucu durumunu gösterir; eylemler [onAction] ile gider (onaylı:
/// kart açma ve oy geri alınamaz). Sonuç ekrana WS olaylarıyla yansır.
class GameBody extends StatefulWidget {
  const GameBody({
    required this.game,
    required this.myUserId,
    this.onAction,
    this.onPlayerMenu,
    this.online = true,
    this.clock = DateTime.now,
    super.key,
  });

  final GameView game;
  final String? myUserId;

  /// Eylemi gönderir; hata gösterimi çağıranındır. `null` → salt okunur.
  final Future<void> Function(GameAction action)? onAction;

  /// Başka bir oyuncunun kutusuna dokunulunca (oylama hedefi değilse).
  final ValueChanged<Participant>? onPlayerMenu;

  /// Bağlantı yokken eylem düğmeleri görünür ama devre dışıdır.
  final bool online;

  /// Cihaz saati (testlerde sahte saat verilir).
  final DateTime Function() clock;

  @override
  State<GameBody> createState() => _GameBodyState();
}

class _GameBodyState extends State<GameBody> {
  /// "Kapı kapandı" gösterim süresi.
  static const doorClosedFor = Duration(seconds: 2);

  bool _busy = false;
  bool _doorClosed = false;
  Timer? _doorTimer;

  GameView get game => widget.game;

  @override
  void didUpdateWidget(GameBody old) {
    super.didUpdateWidget(old);
    final was = old.game.participantOf(widget.myUserId);
    final now = game.participantOf(widget.myUserId);
    // Elendim (ayrılma değil): kısa süre "Kapı kapandı".
    if ((was?.isAlive ?? false) &&
        now?.status == ParticipantStatus.eliminated) {
      _doorTimer?.cancel();
      setState(() => _doorClosed = true);
      _doorTimer = Timer(doorClosedFor, () {
        if (mounted) setState(() => _doorClosed = false);
      });
    }
  }

  @override
  void dispose() {
    _doorTimer?.cancel();
    super.dispose();
  }

  void _showScenario() {
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

  Future<void> _send(GameAction action) async {
    final onAction = widget.onAction;
    if (onAction == null || _busy || !widget.online) return;
    setState(() => _busy = true);
    try {
      await onAction(action);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _reveal(MyCard card) async {
    final l = context.l10n;
    final ok = await showConfirmDialog(
      context,
      title: l.gameRevealConfirmTitle(
        categoryStyle(context, card.category).label,
      ),
      message: l.gameRevealConfirmMessage(card.text),
      confirmLabel: l.gameRevealConfirm,
    );
    if (ok && mounted) await _send(RevealCard(card.category));
  }

  Future<void> _vote(Participant target, {required bool tiebreak}) async {
    final l = context.l10n;
    final ok = await showConfirmDialog(
      context,
      title: l.gameVoteConfirmTitle(target.nickname),
      message: l.gameVoteConfirmMessage,
      confirmLabel: l.gameVoteConfirm,
      destructive: true,
    );
    if (!ok || !mounted) return;
    await _send(tiebreak ? TiebreakVote(target.userId) : Vote(target.userId));
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final me = game.participantOf(widget.myUserId);
    final spectator = me == null || !me.isAlive;
    final intro =
        game.phase == GamePhase.dealing ||
        game.phase == GamePhase.scenarioIntro;
    final interactive = widget.onAction != null && !_busy && widget.online;
    final voteTargets = interactive
        ? {for (final p in game.voteTargets(widget.myUserId)) p.userId}
        : const <String>{};

    final column = Column(
      children: [
        PhaseBar(
          game: game,
          clock: widget.clock,
          onShowScenario: _showScenario,
        ),
        if (spectator && !intro) _SpectatorBanner(text: l.gameSpectatorBanner),
        Expanded(
          child: intro
              ? _IntroView(game: game)
              : game.phase == GamePhase.elimination &&
                    game.lastElimination != null
              ? EliminationPanel(game: game, result: game.lastElimination!)
              : GridView.builder(
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
                      isMe: p.userId == widget.myUserId,
                      isTurn:
                          game.phase == GamePhase.reveal &&
                          p.userId == game.turnUserId,
                      onVote: voteTargets.contains(p.userId)
                          ? () => unawaited(_vote(p, tiebreak: false))
                          : null,
                      onMenu:
                          widget.onPlayerMenu == null ||
                              p.userId == widget.myUserId
                          ? null
                          : () => widget.onPlayerMenu!(p),
                    );
                  },
                ),
        ),
        if (!intro)
          _ActionArea(
            game: game,
            myUserId: widget.myUserId,
            busy: _busy,
            enabled: widget.online,
            interactive: widget.onAction != null,
            onReveal: (c) => unawaited(_reveal(c)),
            onReady: () => unawaited(_send(const DiscussionReady())),
            onTiebreak: (p) => unawaited(_vote(p, tiebreak: true)),
          ),
        _BottomBar(cards: game.myCards),
      ],
    );
    return Stack(
      children: [column, if (_doorClosed) const DoorClosedOverlay()],
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

/// Aşamaya göre bilgi ve eylemler.
class _ActionArea extends StatelessWidget {
  const _ActionArea({
    required this.game,
    required this.myUserId,
    required this.busy,
    required this.enabled,
    required this.interactive,
    required this.onReveal,
    required this.onReady,
    required this.onTiebreak,
  });

  final GameView game;
  final String? myUserId;
  final bool busy;

  /// `false` → düğmeler görünür ama basılamaz (bağlantı yok).
  final bool enabled;
  final bool interactive;
  final ValueChanged<MyCard> onReveal;
  final VoidCallback onReady;
  final ValueChanged<Participant> onTiebreak;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    String nameOf(String? id) => game.participantOf(id)?.nickname ?? '';

    String title;
    final details = <Widget>[];
    Widget text(String s) => Padding(
      padding: const EdgeInsets.only(top: AppSpacing.s1),
      child: Text(s, style: theme.textTheme.bodyMedium),
    );

    switch (game.phase) {
      case GamePhase.reveal:
        final mine = game.turnUserId == myUserId;
        title = mine
            ? l.gameRevealMine
            : l.gameRevealOther(nameOf(game.turnUserId));
        if (mine && interactive) {
          final allowed = {
            for (final c in game.revealableCards(myUserId)) c.category,
          };
          if (game.round == 1) details.add(text(l.gameRevealRound1Hint));
          details.add(
            Padding(
              padding: const EdgeInsets.only(top: AppSpacing.s2),
              child: Wrap(
                spacing: AppSpacing.s2,
                runSpacing: AppSpacing.s2,
                children: [
                  for (final c in game.myCards)
                    if (!c.revealed)
                      Builder(
                        builder: (context) {
                          final style = categoryStyle(context, c.category);
                          return ActionChip(
                            key: ValueKey('reveal-${c.category.name}'),
                            avatar: Icon(
                              style.icon,
                              size: 18,
                              color: style.color,
                            ),
                            label: Text(style.label),
                            tooltip: c.text,
                            onPressed:
                                allowed.contains(c.category) && !busy && enabled
                                ? () => onReveal(c)
                                : null,
                          );
                        },
                      ),
                ],
              ),
            ),
          );
        }
      case GamePhase.discussion:
        title = l.gameDiscussionInfo;
        details.add(
          text(
            l.gameReadyProgress(
              game.readyProgress.done,
              game.readyProgress.total,
            ),
          ),
        );
        if (interactive && game.canDiscussionReady(myUserId)) {
          details.add(
            Padding(
              padding: const EdgeInsets.only(top: AppSpacing.s3),
              child: AppButton.primary(
                label: l.gameReadyButton,
                icon: Icons.check_rounded,
                isLoading: busy,
                onPressed: enabled ? onReady : null,
              ),
            ),
          );
        } else if (game.discussionReady) {
          details.add(text(l.gameReadyDone));
        }
      case GamePhase.voting:
        title = l.gameVotingInfo;
        details.add(
          text(
            l.gameVoteProgress(game.voteProgress.done, game.voteProgress.total),
          ),
        );
        if (game.voted) {
          details.add(text(l.gameVoteDone));
        } else if (interactive && game.voteTargets(myUserId).isNotEmpty) {
          details.add(text(l.gameVoteHint));
        }
      case GamePhase.tieBreak:
        final targets = interactive
            ? game.tiebreakTargets(myUserId)
            : const <Participant>[];
        title = targets.isNotEmpty ? l.gameTieBreakMine : l.gameTieBreakInfo;
        if (game.tiebreakProgress.total > 0) {
          details.add(
            text(
              l.gameVoteProgress(
                game.tiebreakProgress.done,
                game.tiebreakProgress.total,
              ),
            ),
          );
        }
        if (game.tiebreakVoted) details.add(text(l.gameVoteDone));
        if (targets.isNotEmpty) {
          details.add(
            Padding(
              padding: const EdgeInsets.only(top: AppSpacing.s2),
              child: Wrap(
                spacing: AppSpacing.s2,
                children: [
                  for (final p in targets)
                    OutlinedButton(
                      key: ValueKey('tiebreak-${p.userId}'),
                      onPressed: busy || !enabled ? null : () => onTiebreak(p),
                      child: Text(p.nickname),
                    ),
                ],
              ),
            ),
          );
        }
      case GamePhase.elimination:
        // Ayrıntı ızgaranın yerindeki eleme panelinde.
        title = phaseLabel(l, game.phase);
      case GamePhase.finished:
        title = l.gamePhaseFinished;
      case GamePhase.dealing || GamePhase.scenarioIntro:
        title = '';
    }

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
          ...details,
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
