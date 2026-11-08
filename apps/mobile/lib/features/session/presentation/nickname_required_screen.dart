import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/ui/app_buttons.dart';
import 'package:siginak/core/ui/app_text_field.dart';
import 'package:siginak/features/session/domain/nickname_rules.dart';
import 'package:siginak/features/session/presentation/session_controller.dart';

/// `/nickname` — benzersiz kullanıcı adı gelmeden önce açılmış hesaplar
/// adlarını bir kez onaylar (ya da alınmışsa yenisini seçer). Onaylanınca
/// yönlendirme kullanıcıyı ana ekrana/odasına götürür.
class NicknameRequiredScreen extends ConsumerStatefulWidget {
  const NicknameRequiredScreen({super.key});

  @override
  ConsumerState<NicknameRequiredScreen> createState() =>
      _NicknameRequiredScreenState();
}

class _NicknameRequiredScreenState
    extends ConsumerState<NicknameRequiredScreen> {
  late final _controller = TextEditingController(
    text: ref.read(sessionControllerProvider).nickname ?? '',
  );
  bool _saving = false;
  bool _touched = false;
  String? _serverError;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  Future<void> _confirm() async {
    setState(() => _touched = true);
    if (checkNickname(_controller.text) != null || _saving) return;
    setState(() => _saving = true);
    final l = context.l10n;
    try {
      await ref
          .read(sessionControllerProvider.notifier)
          .confirmNickname(_controller.text.trim());
    } on ApiException catch (e) {
      if (!mounted) return;
      final issue = NicknameIssue.fromCode(e.fieldErrors['nickname']);
      setState(
        () => _serverError = issue?.message(l) ?? messageForApiError(l, e),
      );
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final local = _touched ? checkNickname(_controller.text)?.message(l) : null;

    return Scaffold(
      appBar: AppBar(
        automaticallyImplyLeading: false,
        title: Text(l.nicknameRequiredTitle),
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
                    l.nicknameRequiredBody,
                    style: theme.textTheme.bodyLarge,
                  ),
                  const SizedBox(height: AppSpacing.s5),
                  AppTextField(
                    key: const ValueKey('nickname-required-field'),
                    label: l.welcomeNicknameLabel,
                    controller: _controller,
                    enabled: !_saving,
                    maxLength: nicknameMaxLength,
                    errorText: _serverError ?? local,
                    textInputAction: TextInputAction.done,
                    onChanged: (_) => setState(() {
                      _touched = true;
                      _serverError = null;
                    }),
                    onSubmitted: (_) => unawaited(_confirm()),
                  ),
                  const SizedBox(height: AppSpacing.s4),
                  AppButton.primary(
                    key: const ValueKey('nickname-required-confirm'),
                    label: l.nicknameRequiredConfirm,
                    isLoading: _saving,
                    onPressed: () => unawaited(_confirm()),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
