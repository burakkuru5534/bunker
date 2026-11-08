import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/theme/app_typography.dart';
import 'package:siginak/core/ui/app_buttons.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:siginak/core/ui/confirm_dialog.dart';
import 'package:siginak/core/ui/player_avatar.dart';
import 'package:siginak/core/ui/status_badge.dart';
import 'package:siginak/features/room/data/room_models.dart';
import 'package:siginak/features/room/presentation/room_controller.dart';

/// Kapasite seçenekleri (docs/game-design.md §1).
const capacityOptions = [1, 2];

/// Lobi görünümü (docs/ui/screen-map.md §3.4a). Yalnızca gösterir; eylemler
/// geri çağrılarla oda ekranına gider.
class LobbyView extends StatefulWidget {
  const LobbyView({
    required this.state,
    required this.onReady,
    required this.onCapacity,
    required this.onKick,
    required this.onStart,
    this.scenarios = const AsyncValue.data([]),
    this.onScenario,
    this.onRetryScenarios,
    this.onMemberMenu,
    this.onInviteFriends,
    this.online = true,
    super.key,
  });

  final RoomState state;
  final Future<void> Function(bool ready) onReady;
  final Future<void> Function(int capacity) onCapacity;
  final Future<void> Function(String userId) onKick;
  final Future<void> Function() onStart;

  /// Seçilebilir senaryolar (`GET /v1/scenarios`).
  final AsyncValue<List<ScenarioOption>> scenarios;

  /// Oda sahibinin senaryo seçimi; `null` = rastgele.
  final Future<void> Function(String? scenarioId)? onScenario;
  final VoidCallback? onRetryScenarios;

  /// Başka bir üyeye dokunulunca: oyuncu menüsü (Raporla / Engelle).
  final ValueChanged<RoomMember>? onMemberMenu;

  /// "Arkadaşlarını davet et" (arkadaş listesi sayfası).
  final VoidCallback? onInviteFriends;

  /// Bağlantı yokken eylemler görünür ama devre dışıdır.
  final bool online;

  @override
  State<LobbyView> createState() => _LobbyViewState();
}

class _LobbyViewState extends State<LobbyView> {
  /// Süren istek (aynı düğmeye tekrar basılmasın).
  String? _busy;

  /// Yeni eylem başlatılabilir: süren istek yok ve bağlıyız.
  bool get _idle => _busy == null && widget.online;

  Future<void> _guard(String key, Future<void> Function() action) async {
    if (!_idle) return;
    setState(() => _busy = key);
    try {
      await action();
    } finally {
      if (mounted) setState(() => _busy = null);
    }
  }

  Future<void> _confirmKick(RoomMember member) async {
    final l = context.l10n;
    final ok = await showConfirmDialog(
      context,
      title: l.roomKickTitle(member.nickname),
      message: l.roomKickMessage,
      confirmLabel: l.roomKickConfirm,
      destructive: true,
    );
    if (ok && mounted) {
      await _guard('kick:${member.userId}', () => widget.onKick(member.userId));
    }
  }

  Future<void> _copyCode(String code) async {
    await Clipboard.setData(ClipboardData(text: code));
    if (mounted) showAppSnackBar(context, context.l10n.roomCopied);
  }

  String? _startReason(AppLocalizations l, RoomView room) =>
      switch (startBlockerOf(room)) {
        null => null,
        TooFewPlayers(:final count) => l.roomStartTooFew(count, minPlayers),
        TooManyPlayers() => l.roomStartTooMany(maxPlayers),
        NotAllReady(:final ready, :final total) => l.roomStartNotReady(
          ready,
          total,
        ),
      };

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final state = widget.state;
    final room = state.room!;
    final me = state.me;
    final isHost = state.isHost;
    final startReason = _startReason(l, room);

    return Column(
      children: [
        Expanded(
          child: ListView(
            padding: const EdgeInsets.all(AppSpacing.screen),
            children: [
              _InviteCodeCard(
                code: room.inviteCode,
                onCopy: () => unawaited(_copyCode(room.inviteCode)),
              ),
              if (widget.onInviteFriends != null) ...[
                const SizedBox(height: AppSpacing.s3),
                AppButton.secondary(
                  key: const ValueKey('lobby-invite-friends'),
                  label: l.roomInviteFriends,
                  icon: Icons.group_add_outlined,
                  onPressed: widget.online ? widget.onInviteFriends : null,
                ),
              ],
              const SizedBox(height: AppSpacing.s4),
              _CapacityCard(
                capacity: room.capacity,
                editable: isHost && _idle,
                onChanged: (c) =>
                    unawaited(_guard('capacity', () => widget.onCapacity(c))),
              ),
              const SizedBox(height: AppSpacing.s4),
              _ScenarioCard(
                selected: room.scenarioId,
                options: widget.scenarios,
                editable: isHost && _idle && widget.onScenario != null,
                onChanged: (id) =>
                    unawaited(_guard('scenario', () => widget.onScenario!(id))),
                onRetry: widget.onRetryScenarios,
              ),
              const SizedBox(height: AppSpacing.s5),
              Text(
                l.roomPlayers(room.players.length, maxPlayers),
                style: theme.textTheme.titleMedium,
              ),
              const SizedBox(height: AppSpacing.s2),
              for (final (i, m) in room.members.indexed)
                _MemberTile(
                  member: m,
                  seat: i,
                  isHost: m.userId == room.hostUserId,
                  isMe: m.userId == state.myUserId,
                  onKick: isHost && m.userId != state.myUserId && _idle
                      ? () => unawaited(_confirmKick(m))
                      : null,
                  onTap:
                      widget.onMemberMenu == null || m.userId == state.myUserId
                      ? null
                      : () => widget.onMemberMenu!(m),
                ),
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
                if (me != null && me.role == MemberRole.player)
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    title: Text(l.roomReady),
                    value: me.isReady,
                    onChanged: _idle
                        ? (v) => unawaited(
                            _guard('ready', () => widget.onReady(v)),
                          )
                        : null,
                  ),
                if (isHost)
                  AppButton.primary(
                    label: l.roomStart,
                    icon: Icons.play_arrow_rounded,
                    isLoading: _busy == 'start',
                    disabledReason: startReason,
                    onPressed: startReason == null && _idle
                        ? () => unawaited(_guard('start', widget.onStart))
                        : null,
                  )
                else
                  Text(
                    l.roomWaitingHost,
                    textAlign: TextAlign.center,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
              ],
            ),
          ),
        ),
      ],
    );
  }
}

class _InviteCodeCard extends StatelessWidget {
  const _InviteCodeCard({required this.code, required this.onCopy});

  final String code;
  final VoidCallback onCopy;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final codeBlock = Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(l.roomInviteCode, style: theme.textTheme.labelLarge),
        const SizedBox(height: AppSpacing.s1),
        // Kod tek satırda kalır; sığmazsa küçülür (bölünmüş kod okunmaz).
        FittedBox(
          fit: BoxFit.scaleDown,
          alignment: AlignmentDirectional.centerStart,
          child: SelectableText(
            code,
            style: theme.textTheme.headlineLarge?.copyWith(
              fontFamily: AppFonts.display,
              letterSpacing: 4,
            ),
          ),
        ),
      ],
    );
    final copy = AppButton.secondary(
      label: l.roomCopy,
      icon: Icons.copy_rounded,
      expand: false,
      onPressed: onCopy,
    );
    // Büyük yazı ölçeğinde veya dar ekranda kod ve düğme alt alta
    // (PLAY-012: 320 dp, %200 yazı).
    final stacked = MediaQuery.textScalerOf(context).scale(1) > 1.3;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.s4),
        child: stacked
            ? Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  codeBlock,
                  const SizedBox(height: AppSpacing.s3),
                  copy,
                ],
              )
            : Row(
                children: [
                  Expanded(child: codeBlock),
                  const SizedBox(width: AppSpacing.s2),
                  copy,
                ],
              ),
      ),
    );
  }
}

class _CapacityCard extends StatelessWidget {
  const _CapacityCard({
    required this.capacity,
    required this.editable,
    required this.onChanged,
  });

  final int capacity;
  final bool editable;
  final ValueChanged<int> onChanged;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.s4),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(l.roomCapacity, style: theme.textTheme.labelLarge),
            const SizedBox(height: AppSpacing.s1),
            Text(l.roomCapacityHelp, style: theme.textTheme.bodySmall),
            const SizedBox(height: AppSpacing.s3),
            SegmentedButton<int>(
              segments: [
                for (final c in capacityOptions)
                  ButtonSegment(value: c, label: Text(l.roomCapacityValue(c))),
              ],
              selected: {capacity},
              showSelectedIcon: false,
              onSelectionChanged: editable
                  ? (s) {
                      if (s.single != capacity) onChanged(s.single);
                    }
                  : null,
            ),
          ],
        ),
      ),
    );
  }
}

/// Senaryo seçimi: oda sahibi listeden seçer, diğerleri yalnızca görür.
class _ScenarioCard extends StatelessWidget {
  const _ScenarioCard({
    required this.selected,
    required this.options,
    required this.editable,
    required this.onChanged,
    required this.onRetry,
  });

  final String? selected;
  final AsyncValue<List<ScenarioOption>> options;
  final bool editable;
  final ValueChanged<String?> onChanged;
  final VoidCallback? onRetry;

  String _label(AppLocalizations l) {
    final id = selected;
    if (id == null) return l.roomScenarioRandom;
    for (final o in options.value ?? const <ScenarioOption>[]) {
      if (o.id == id) return o.title;
    }
    return options.isLoading ? l.roomScenarioLoading : id;
  }

  Future<void> _pick(BuildContext context) async {
    final list = options.value;
    if (list == null) return;
    final l = context.l10n;
    // İç içe nullable: dış `null` = vazgeçti, iç `null` = rastgele.
    final choice = await showModalBottomSheet<({String? id})>(
      context: context,
      showDragHandle: true,
      isScrollControlled: true,
      builder: (context) => SafeArea(
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: AppSpacing.screen,
                ),
                child: Text(
                  l.roomScenarioChoose,
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              ),
              const SizedBox(height: AppSpacing.s2),
              for (final (id, title) in [
                (null, l.roomScenarioRandom),
                for (final o in list) (o.id, o.title),
              ])
                ListTile(
                  key: ValueKey('scenario-option-${id ?? 'random'}'),
                  leading: Icon(
                    id == selected
                        ? Icons.radio_button_checked_rounded
                        : Icons.radio_button_unchecked_rounded,
                  ),
                  title: Text(title),
                  selected: id == selected,
                  onTap: () => Navigator.of(context).pop((id: id)),
                ),
            ],
          ),
        ),
      ),
    );
    if (choice != null && choice.id != selected) onChanged(choice.id);
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final failed = options.hasError && options.value == null;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(AppSpacing.s4),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(l.roomScenario, style: theme.textTheme.labelLarge),
            const SizedBox(height: AppSpacing.s1),
            Text(l.roomScenarioHelp, style: theme.textTheme.bodySmall),
            const SizedBox(height: AppSpacing.s3),
            if (editable && failed)
              Row(
                children: [
                  Expanded(
                    child: Text(
                      l.roomScenarioLoadFailed,
                      style: theme.textTheme.bodyMedium,
                    ),
                  ),
                  TextButton(onPressed: onRetry, child: Text(l.actionRetry)),
                ],
              )
            else
              OutlinedButton.icon(
                key: const ValueKey('scenario-picker'),
                onPressed: editable && options.value != null
                    ? () => unawaited(_pick(context))
                    : null,
                icon: const Icon(Icons.public_rounded),
                label: Text(
                  _label(l),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
          ],
        ),
      ),
    );
  }
}

class _MemberTile extends StatelessWidget {
  const _MemberTile({
    required this.member,
    required this.seat,
    required this.isHost,
    required this.isMe,
    required this.onKick,
    this.onTap,
  });

  final RoomMember member;
  final int seat;
  final bool isHost;
  final bool isMe;
  final VoidCallback? onKick;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final badges = [
      if (isHost) StatusBadgeKind.host,
      if (member.role == MemberRole.spectator)
        StatusBadgeKind.spectator
      else if (member.isReady)
        StatusBadgeKind.ready
      else
        StatusBadgeKind.waiting,
    ];
    return ListTile(
      key: ValueKey('member-${member.userId}'),
      contentPadding: EdgeInsets.zero,
      onTap: onTap,
      // Lobide bağlantı olayı yok (api.md §5); `connected` yalnızca
      // snapshot anını gösterir, bu yüzden lobide durum gösterilmez.
      leading: PlayerAvatar(nickname: member.nickname, seat: seat),
      title: Text(isMe ? '${member.nickname} (${l.roomYou})' : member.nickname),
      subtitle: Padding(
        padding: const EdgeInsets.only(top: AppSpacing.s1),
        child: Wrap(
          spacing: AppSpacing.s2,
          children: [for (final b in badges) StatusBadge(b)],
        ),
      ),
      trailing: onKick == null
          ? null
          : IconButton(
              tooltip: l.roomKick,
              onPressed: onKick,
              icon: const Icon(Icons.person_remove_outlined),
            ),
    );
  }
}
