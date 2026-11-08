import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/core/config/app_config_provider.dart';
import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/ui/app_buttons.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:siginak/core/ui/brand_logo.dart';
import 'package:siginak/features/friends/friends.dart';
import 'package:siginak/features/room/room.dart';
import 'package:siginak/features/session/session.dart';

/// `ALREADY_IN_ROOM` hatasındaki mevcut oda (docs/api.md "Oda").
String? existingRoomOf(ApiException e) => switch (e.details['room_id']) {
  final String id when e.code == 'ALREADY_IN_ROOM' => id,
  _ => null,
};

/// Ortam/adres etiketi yalnızca geliştirme (debug) derlemesinde ve üretim
/// dışı ortamda görünür. Release derlemesinde (test sunucusuna bağlı olsa da;
/// ör. mağaza ekran görüntüleri, kapalı test) gösterilmez.
bool showEnvironmentLabel(AppConfig config, {bool debugBuild = kDebugMode}) =>
    debugBuild && !config.isProduction;

/// `/home` — ana ekran (docs/ui/screen-map.md §3.2).
///
/// Gezinme geri çağrılarla verilir; ekran rota yollarını bilmez.
class HomeScreen extends ConsumerStatefulWidget {
  const HomeScreen({
    required this.onJoin,
    required this.onOpenSettings,
    required this.onOpenRoom,
    required this.onOpenFriends,
    required this.onJoinInvite,
    super.key,
  });

  final VoidCallback onJoin;
  final VoidCallback onOpenSettings;
  final VoidCallback onOpenFriends;

  /// Davet kartındaki "Katıl" → davet koduyla katılma ekranı.
  final ValueChanged<String> onJoinInvite;

  /// Oda kuruldu (veya kullanıcı zaten bir odadaydı) → oda ekranına geç.
  final ValueChanged<String> onOpenRoom;

  @override
  ConsumerState<HomeScreen> createState() => _HomeScreenState();
}

/// Ana ekran açıkken davet/istek kutusunun yoklanma aralığı.
const inboxPollInterval = Duration(seconds: 20);

class _HomeScreenState extends ConsumerState<HomeScreen> {
  bool _creating = false;
  Timer? _poll;
  late final AppLifecycleListener _lifecycle;

  /// Reddedilen davetler, sunucu yanıtı gelene kadar hemen gizlenir.
  final _declined = <String>{};

  @override
  void initState() {
    super.initState();
    _startPolling();
    _lifecycle = AppLifecycleListener(
      onShow: () {
        ref.invalidate(inboxProvider);
        _startPolling();
      },
      onHide: _stopPolling,
    );
  }

  @override
  void dispose() {
    _stopPolling();
    _lifecycle.dispose();
    super.dispose();
  }

  void _startPolling() {
    _poll?.cancel();
    _poll = Timer.periodic(
      inboxPollInterval,
      (_) => ref.invalidate(inboxProvider),
    );
  }

  void _stopPolling() {
    _poll?.cancel();
    _poll = null;
  }

  Future<void> _decline(RoomInvite invite) async {
    setState(() => _declined.add(invite.roomId));
    try {
      await ref.read(friendsRepositoryProvider).decline(invite.roomId);
    } on ApiException {
      // Davet zaten düşmüş olabilir; bir sonraki yoklama durumu düzeltir.
    }
    ref.invalidate(inboxProvider);
  }

  void _openRoom(String roomId) {
    ref.read(sessionControllerProvider.notifier).setActiveRoom(roomId);
    widget.onOpenRoom(roomId);
  }

  Future<void> _create() async {
    if (_creating) return;
    setState(() => _creating = true);
    try {
      final room = await ref.read(roomRepositoryProvider).create();
      if (mounted) _openRoom(room.id);
    } on ApiException catch (e) {
      if (!mounted) return;
      final l = context.l10n;
      final existing = existingRoomOf(e);
      final message = messageForApiError(l, e);
      if (message != null) {
        showAppSnackBar(
          context,
          message,
          isError: true,
          actionLabel: existing != null ? l.homeBackToRoom : null,
          onAction: existing != null ? () => _openRoom(existing) : null,
        );
      }
    } finally {
      if (mounted) setState(() => _creating = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final config = ref.watch(appConfigProvider);
    final nickname = ref.watch(
      sessionControllerProvider.select((s) => s.nickname),
    );
    final theme = Theme.of(context);
    final textTheme = theme.textTheme;
    // Yoklama hatası ekranı bozmaz; son başarılı yanıt gösterilmeye devam eder.
    final inbox = ref.watch(inboxProvider).value ?? Inbox.empty;
    final requests = inbox.incomingRequests;
    final invites = [
      for (final i in inbox.invites)
        if (!_declined.contains(i.roomId)) i,
    ];

    return Scaffold(
      appBar: AppBar(
        actions: [
          IconButton(
            key: const ValueKey('home-friends'),
            tooltip: requests > 0
                ? l.friendsOpenWithRequests(requests)
                : l.friendsOpen,
            onPressed: widget.onOpenFriends,
            icon: Badge(
              isLabelVisible: requests > 0,
              label: Text('$requests'),
              child: const Icon(Icons.people_alt_outlined),
            ),
          ),
          IconButton(
            tooltip: l.actionSettings,
            onPressed: widget.onOpenSettings,
            icon: const Icon(Icons.settings_outlined),
          ),
        ],
      ),
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(AppSpacing.screen),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 480),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Center(child: BrandLogo(maxWidth: 280)),
                  if (nickname != null) ...[
                    const SizedBox(height: AppSpacing.s4),
                    Text(
                      l.homeGreeting(nickname),
                      style: textTheme.bodyLarge,
                      textAlign: TextAlign.center,
                    ),
                  ],
                  for (final invite in invites) ...[
                    const SizedBox(height: AppSpacing.s4),
                    _InviteCard(
                      key: ValueKey('invite-${invite.roomId}'),
                      invite: invite,
                      onJoin: () => widget.onJoinInvite(invite.inviteCode),
                      onDecline: () => unawaited(_decline(invite)),
                    ),
                  ],
                  const SizedBox(height: AppSpacing.s6),
                  AppButton.primary(
                    label: l.homeCreateRoom,
                    icon: Icons.add_rounded,
                    isLoading: _creating,
                    onPressed: () => unawaited(_create()),
                  ),
                  const SizedBox(height: AppSpacing.s3),
                  AppButton.secondary(
                    label: l.homeJoinRoom,
                    icon: Icons.login_rounded,
                    onPressed: _creating ? null : widget.onJoin,
                  ),
                  const SizedBox(height: AppSpacing.s5),
                  const _HowToPlay(),
                  if (showEnvironmentLabel(config)) ...[
                    const SizedBox(height: AppSpacing.s5),
                    Text(
                      'Ortam: ${config.environment.name} · ${config.apiBaseUrl}',
                      style: textTheme.bodySmall,
                      textAlign: TextAlign.center,
                    ),
                  ],
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _InviteCard extends StatelessWidget {
  const _InviteCard({
    required this.invite,
    required this.onJoin,
    required this.onDecline,
    super.key,
  });

  final RoomInvite invite;
  final VoidCallback onJoin;
  final VoidCallback onDecline;

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
            Row(
              children: [
                Icon(
                  Icons.mail_outline_rounded,
                  color: theme.colorScheme.primary,
                ),
                const SizedBox(width: AppSpacing.s2),
                Expanded(
                  child: Text(
                    l.inviteFrom(invite.from.nickname),
                    style: theme.textTheme.titleSmall,
                  ),
                ),
              ],
            ),
            const SizedBox(height: AppSpacing.s1),
            Text(
              l.invitePlayers(invite.players),
              style: theme.textTheme.bodyMedium,
            ),
            const SizedBox(height: AppSpacing.s2),
            Wrap(
              alignment: WrapAlignment.end,
              spacing: AppSpacing.s2,
              children: [
                TextButton(
                  key: const ValueKey('invite-decline'),
                  onPressed: onDecline,
                  child: Text(l.inviteDecline),
                ),
                FilledButton(
                  key: const ValueKey('invite-join'),
                  onPressed: onJoin,
                  child: Text(l.inviteJoin),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _HowToPlay extends StatelessWidget {
  const _HowToPlay();

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final rules = [
      l.homeHowTo1,
      l.homeHowTo2,
      l.homeHowTo3,
      l.homeHowTo4,
      l.homeHowTo5,
    ];
    return Card(
      clipBehavior: Clip.antiAlias,
      child: ExpansionTile(
        leading: const Icon(Icons.help_outline_rounded),
        title: Text(l.homeHowToTitle),
        childrenPadding: const EdgeInsets.fromLTRB(
          AppSpacing.s4,
          0,
          AppSpacing.s4,
          AppSpacing.s4,
        ),
        expandedCrossAxisAlignment: CrossAxisAlignment.start,
        children: [
          for (var i = 0; i < rules.length; i++)
            Padding(
              padding: const EdgeInsets.only(bottom: AppSpacing.s2),
              child: Text(
                '${i + 1}. ${rules[i]}',
                style: theme.textTheme.bodyMedium,
              ),
            ),
        ],
      ),
    );
  }
}
