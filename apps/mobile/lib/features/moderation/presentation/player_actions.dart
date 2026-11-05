import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:siginak/core/ui/confirm_dialog.dart';
import 'package:siginak/features/moderation/data/moderation_repository.dart';

/// Oyuncu menüsü: **Raporla** / **Engelle** (docs/ui/screen-map.md §3.4c).
///
/// [onBlocked]: engelleme başarılı olunca (ör. sohbetteki mesajlarını gizle).
Future<void> showPlayerActions(
  BuildContext context, {
  required String userId,
  required String nickname,
  required String roomId,
  VoidCallback? onBlocked,
}) => showModalBottomSheet<void>(
  context: context,
  showDragHandle: true,
  builder: (sheet) {
    final l = sheet.l10n;
    return SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: AppSpacing.screen),
            child: Text(nickname, style: Theme.of(sheet).textTheme.titleLarge),
          ),
          ListTile(
            leading: const Icon(Icons.flag_outlined),
            title: Text(l.modReport),
            onTap: () {
              Navigator.of(sheet).pop();
              unawaited(
                showDialog<void>(
                  context: context,
                  builder: (_) => _ReportDialog(
                    userId: userId,
                    nickname: nickname,
                    roomId: roomId,
                  ),
                ),
              );
            },
          ),
          ListTile(
            leading: const Icon(Icons.block_rounded),
            title: Text(l.modBlock),
            onTap: () {
              Navigator.of(sheet).pop();
              unawaited(_block(context, userId, nickname, onBlocked));
            },
          ),
        ],
      ),
    );
  },
);

Future<void> _block(
  BuildContext context,
  String userId,
  String nickname,
  VoidCallback? onBlocked,
) async {
  final l = context.l10n;
  final ok = await showConfirmDialog(
    context,
    title: l.modBlockTitle(nickname),
    message: l.modBlockMessage,
    confirmLabel: l.modBlock,
    destructive: true,
  );
  if (!ok || !context.mounted) return;
  try {
    await ProviderScope.containerOf(context)
        .read(moderationRepositoryProvider)
        .block(userId);
    if (!context.mounted) return;
    onBlocked?.call();
    showAppSnackBar(context, l.modBlocked(nickname));
  } on ApiException catch (e) {
    if (!context.mounted) return;
    final message = messageForApiError(l, e);
    if (message != null) showAppSnackBar(context, message, isError: true);
  }
}

class _ReportDialog extends ConsumerStatefulWidget {
  const _ReportDialog({
    required this.userId,
    required this.nickname,
    required this.roomId,
  });

  final String userId;
  final String nickname;
  final String roomId;

  @override
  ConsumerState<_ReportDialog> createState() => _ReportDialogState();
}

class _ReportDialogState extends ConsumerState<_ReportDialog> {
  final _note = TextEditingController();
  ReportReason? _reason;
  bool _sending = false;
  String? _error;

  @override
  void dispose() {
    _note.dispose();
    super.dispose();
  }

  Future<void> _send() async {
    final reason = _reason;
    if (reason == null || _sending) return;
    setState(() {
      _sending = true;
      _error = null;
    });
    final l = context.l10n;
    try {
      await ref
          .read(moderationRepositoryProvider)
          .report(
            reportedUserId: widget.userId,
            roomId: widget.roomId,
            reason: reason,
            note: _note.text,
          );
      if (!mounted) return;
      Navigator.of(context).pop();
      showAppSnackBar(context, l.modReportSent);
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _sending = false;
        _error = messageForApiError(l, e);
      });
    }
  }

  static String reasonLabel(AppLocalizations l, ReportReason r) => switch (r) {
    ReportReason.harassment => l.modReasonHarassment,
    ReportReason.hate => l.modReasonHate,
    ReportReason.spam => l.modReasonSpam,
    ReportReason.cheating => l.modReasonCheating,
    ReportReason.other => l.modReasonOther,
  };

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    return AlertDialog(
      title: Text(l.modReportTitle(widget.nickname)),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            RadioGroup<ReportReason>(
              groupValue: _reason,
              onChanged: (r) {
                if (!_sending) setState(() => _reason = r);
              },
              child: Column(
                children: [
                  for (final r in ReportReason.values)
                    RadioListTile<ReportReason>(
                      key: ValueKey('reason-${r.name}'),
                      contentPadding: EdgeInsets.zero,
                      value: r,
                      title: Text(reasonLabel(l, r)),
                    ),
                ],
              ),
            ),
            TextField(
              controller: _note,
              enabled: !_sending,
              maxLength: 200,
              maxLines: 3,
              minLines: 1,
              decoration: InputDecoration(labelText: l.modReportNote),
            ),
            if (_error != null)
              Text(
                _error!,
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.error,
                ),
              ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: _sending ? null : () => Navigator.of(context).pop(),
          child: Text(l.actionCancel),
        ),
        TextButton(
          onPressed: _reason == null || _sending
              ? null
              : () => unawaited(_send()),
          child: Text(l.modReportSend),
        ),
      ],
    );
  }
}
