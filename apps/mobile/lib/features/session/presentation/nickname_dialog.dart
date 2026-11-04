import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/ui/app_text_field.dart';
import 'package:siginak/features/session/domain/nickname_rules.dart';
import 'package:siginak/features/session/presentation/session_controller.dart';

/// Takma ad değiştirme penceresi (`PATCH /v1/me`). Değiştiyse `true`.
Future<bool> showChangeNicknameDialog(BuildContext context) async {
  final changed = await showDialog<bool>(
    context: context,
    builder: (_) => const _NicknameDialog(),
  );
  return changed ?? false;
}

class _NicknameDialog extends ConsumerStatefulWidget {
  const _NicknameDialog();

  @override
  ConsumerState<_NicknameDialog> createState() => _NicknameDialogState();
}

class _NicknameDialogState extends ConsumerState<_NicknameDialog> {
  late final _controller = TextEditingController(
    text: ref.read(sessionControllerProvider).nickname ?? '',
  );
  bool _saving = false;
  bool _touched = false;

  /// Sunucu hatası: alan hatası (ör. ayrılmış ad) ya da genel mesaj.
  String? _serverError;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    setState(() => _touched = true);
    if (checkNickname(_controller.text) != null || _saving) return;
    setState(() => _saving = true);
    final l = context.l10n;
    try {
      await ref
          .read(sessionControllerProvider.notifier)
          .changeNickname(_controller.text.trim());
      if (mounted) Navigator.of(context).pop(true);
    } on ApiException catch (e) {
      if (!mounted) return;
      final issue = NicknameIssue.fromCode(e.fieldErrors['nickname']);
      setState(() {
        _saving = false;
        _serverError = issue?.message(l) ?? messageForApiError(l, e);
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final local = _touched ? checkNickname(_controller.text)?.message(l) : null;
    return AlertDialog(
      title: Text(l.nicknameDialogTitle),
      content: AppTextField(
        label: l.welcomeNicknameLabel,
        controller: _controller,
        autofocus: true,
        enabled: !_saving,
        maxLength: nicknameMaxLength,
        errorText: _serverError ?? local,
        textInputAction: TextInputAction.done,
        onChanged: (_) => setState(() {
          _touched = true;
          _serverError = null;
        }),
        onSubmitted: (_) => unawaited(_save()),
      ),
      actions: [
        TextButton(
          onPressed: _saving ? null : () => Navigator.of(context).pop(false),
          child: Text(l.actionCancel),
        ),
        TextButton(
          onPressed: _saving ? null : () => unawaited(_save()),
          child: Text(l.actionSave),
        ),
      ],
    );
  }
}
