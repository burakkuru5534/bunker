import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/config/app_config_provider.dart';
import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/ui/app_buttons.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:siginak/features/room/room.dart';
import 'package:siginak/features/session/session.dart';

/// `ALREADY_IN_ROOM` hatasındaki mevcut oda (docs/api.md "Oda").
String? existingRoomOf(ApiException e) => switch (e.details['room_id']) {
  final String id when e.code == 'ALREADY_IN_ROOM' => id,
  _ => null,
};

/// `/home` — ana ekran (docs/ui/screen-map.md §3.2).
///
/// Gezinme geri çağrılarla verilir; ekran rota yollarını bilmez.
class HomeScreen extends ConsumerStatefulWidget {
  const HomeScreen({
    required this.onJoin,
    required this.onOpenSettings,
    required this.onOpenRoom,
    super.key,
  });

  final VoidCallback onJoin;
  final VoidCallback onOpenSettings;

  /// Oda kuruldu (veya kullanıcı zaten bir odadaydı) → oda ekranına geç.
  final ValueChanged<String> onOpenRoom;

  @override
  ConsumerState<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends ConsumerState<HomeScreen> {
  bool _creating = false;

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

    return Scaffold(
      appBar: AppBar(
        actions: [
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
                  Text(
                    'SIĞINAK',
                    style: textTheme.displayMedium,
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: AppSpacing.s2),
                  Text(
                    'Son Kalanlar',
                    style: textTheme.titleMedium,
                    textAlign: TextAlign.center,
                  ),
                  if (nickname != null) ...[
                    const SizedBox(height: AppSpacing.s4),
                    Text(
                      l.homeGreeting(nickname),
                      style: textTheme.bodyLarge,
                      textAlign: TextAlign.center,
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
                  if (!config.isProduction) ...[
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
