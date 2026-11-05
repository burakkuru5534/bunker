import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/config/app_version.dart';
import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/theme/game_theme.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:siginak/core/ui/confirm_dialog.dart';
import 'package:siginak/features/session/session.dart';

/// `/settings` — takma ad, engellenenler, hesap silme, sürüm
/// (docs/ui/screen-map.md §3.5). Gizlilik politikası ve kullanım koşulları
/// bağlantıları SAFE-008 ile eklenecek.
class SettingsScreen extends ConsumerStatefulWidget {
  const SettingsScreen({required this.onOpenBlockedUsers, super.key});

  final VoidCallback onOpenBlockedUsers;

  @override
  ConsumerState<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends ConsumerState<SettingsScreen> {
  bool _deleting = false;

  /// İki adımlı onay → `DELETE /v1/me` → oturum kapanır (yönlendirme
  /// hoş geldin ekranına geçer).
  Future<void> _deleteAccount() async {
    final l = context.l10n;
    final first = await showConfirmDialog(
      context,
      title: l.settingsDeleteTitle,
      message: l.settingsDeleteMessage,
      confirmLabel: l.settingsDeleteContinue,
      destructive: true,
    );
    if (!first || !mounted) return;
    final second = await showConfirmDialog(
      context,
      title: l.settingsDeleteFinalTitle,
      message: l.settingsDeleteFinalMessage,
      confirmLabel: l.settingsDeleteAccount,
      destructive: true,
    );
    if (!second || !mounted) return;
    setState(() => _deleting = true);
    try {
      await ref.read(sessionControllerProvider.notifier).deleteAccount();
    } on ApiException catch (e) {
      if (!mounted) return;
      final message = messageForApiError(l, e);
      if (message != null) showAppSnackBar(context, message, isError: true);
    } finally {
      if (mounted) setState(() => _deleting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final nickname = ref.watch(
      sessionControllerProvider.select((s) => s.nickname),
    );
    final danger = context.gameTheme.danger;

    return Scaffold(
      appBar: AppBar(title: Text(l.settingsTitle)),
      body: SafeArea(
        child: ListView(
          children: [
            ListTile(
              leading: const Icon(Icons.badge_outlined),
              title: Text(l.settingsNickname),
              subtitle: Text(nickname ?? ''),
              trailing: const Icon(Icons.edit_outlined),
              onTap: () => unawaited(showChangeNicknameDialog(context)),
            ),
            ListTile(
              leading: const Icon(Icons.block_rounded),
              title: Text(l.blocksTitle),
              trailing: const Icon(Icons.chevron_right_rounded),
              onTap: widget.onOpenBlockedUsers,
            ),
            const Divider(),
            Padding(
              padding: const EdgeInsets.fromLTRB(
                AppSpacing.s4,
                AppSpacing.s3,
                AppSpacing.s4,
                0,
              ),
              child: Text(l.settingsAccount, style: theme.textTheme.labelLarge),
            ),
            ListTile(
              key: const ValueKey('delete-account'),
              leading: _deleting
                  ? const SizedBox.square(
                      dimension: 24,
                      child: CircularProgressIndicator(strokeWidth: 2.5),
                    )
                  : Icon(Icons.delete_forever_outlined, color: danger),
              title: Text(
                l.settingsDeleteAccount,
                style: TextStyle(color: danger),
              ),
              enabled: !_deleting,
              onTap: () => unawaited(_deleteAccount()),
            ),
            const Divider(),
            ListTile(
              leading: const Icon(Icons.info_outline_rounded),
              title: Text(l.settingsVersion(kAppVersion)),
            ),
          ],
        ),
      ),
    );
  }
}
