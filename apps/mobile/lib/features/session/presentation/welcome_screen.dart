import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/config/legal_links.dart';
import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/ui/app_buttons.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:siginak/core/ui/app_text_field.dart';
import 'package:siginak/core/ui/external_link.dart';
import 'package:siginak/features/session/domain/nickname_rules.dart';
import 'package:siginak/features/session/presentation/session_controller.dart';

/// `/welcome` — takma ad ve misafir oturumu (docs/ui/screen-map.md §3.1).
///
/// Başarılı girişte oturum durumu değişir; yönlendirme `/home`'a geçer.
class WelcomeScreen extends ConsumerStatefulWidget {
  const WelcomeScreen({super.key});

  @override
  ConsumerState<WelcomeScreen> createState() => _WelcomeScreenState();
}

class _WelcomeScreenState extends ConsumerState<WelcomeScreen> {
  final _controller = TextEditingController();
  bool _submitting = false;

  /// Kullanıcı yazmaya başlayınca (veya göndermeyi deneyince) hata gösterilir.
  bool _touched = false;

  /// Sunucunun alan hatası (ör. ayrılmış ad); metin değişince silinir.
  NicknameIssue? _serverIssue;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  NicknameIssue? get _localIssue => checkNickname(_controller.text);

  Future<void> _submit() async {
    setState(() => _touched = true);
    if (_localIssue != null || _submitting) return;
    setState(() => _submitting = true);
    try {
      await ref
          .read(sessionControllerProvider.notifier)
          .signInAsGuest(_controller.text.trim());
    } on ApiException catch (e) {
      if (!mounted) return;
      final issue = NicknameIssue.fromCode(e.fieldErrors['nickname']);
      if (issue != null) {
        setState(() => _serverIssue = issue);
      } else {
        final message = messageForApiError(context.l10n, e);
        if (message != null) {
          showAppSnackBar(context, message, isError: true);
        }
      }
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = context.l10n;
    final theme = Theme.of(context);
    final issue = _serverIssue ?? (_touched ? _localIssue : null);

    return Scaffold(
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
                    l.welcomeTitle,
                    style: theme.textTheme.headlineMedium,
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: AppSpacing.s5),
                  for (final rule in [
                    l.welcomeRule1,
                    l.welcomeRule2,
                    l.welcomeRule3,
                  ])
                    _RuleLine(text: rule),
                  const SizedBox(height: AppSpacing.s6),
                  AppTextField(
                    label: l.welcomeNicknameLabel,
                    controller: _controller,
                    helperText: l.welcomeNicknameHelper,
                    errorText: issue?.message(l),
                    maxLength: nicknameMaxLength,
                    enabled: !_submitting,
                    textInputAction: TextInputAction.done,
                    onChanged: (_) => setState(() {
                      _touched = true;
                      _serverIssue = null;
                    }),
                    onSubmitted: (_) => unawaited(_submit()),
                  ),
                  const SizedBox(height: AppSpacing.s4),
                  AppButton.primary(
                    label: l.welcomeStart,
                    isLoading: _submitting,
                    onPressed: () => unawaited(_submit()),
                  ),
                  const SizedBox(height: AppSpacing.s5),
                  Row(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Icon(
                        Icons.info_outline_rounded,
                        size: 18,
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                      const SizedBox(width: AppSpacing.s2),
                      Expanded(
                        child: Text(
                          l.welcomeDeviceNote,
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: AppSpacing.s4),
                  Text(
                    l.welcomeLegalNote,
                    textAlign: TextAlign.center,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                  Wrap(
                    alignment: WrapAlignment.center,
                    children: [
                      TextButton(
                        onPressed: () => unawaited(
                          openExternalLink(context, ref, LegalLinks.terms),
                        ),
                        child: Text(l.legalTerms),
                      ),
                      TextButton(
                        onPressed: () => unawaited(
                          openExternalLink(context, ref, LegalLinks.privacy),
                        ),
                        child: Text(l.legalPrivacy),
                      ),
                    ],
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

class _RuleLine extends StatelessWidget {
  const _RuleLine({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(bottom: AppSpacing.s2),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(
            Icons.chevron_right_rounded,
            size: 20,
            color: theme.colorScheme.primary,
          ),
          const SizedBox(width: AppSpacing.s2),
          Expanded(child: Text(text, style: theme.textTheme.bodyMedium)),
        ],
      ),
    );
  }
}
