import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/error/error_messages.dart';
import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/ui/state_views.dart';
import 'package:siginak/features/session/session.dart';

/// `/` — açılışta oturum kontrol edilirken görünür. Saklı oturum sunucuyla
/// doğrulanamazsa (ağ yok, sunucu hatası) hata ve "Tekrar dene" gösterir.
class SplashScreen extends ConsumerWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final error = ref.watch(
      sessionControllerProvider.select((s) => s.restoreError),
    );
    final l = context.l10n;
    return Scaffold(
      body: SafeArea(
        child: error == null
            ? LoadingView(message: l.splashChecking)
            : ErrorView(
                message: messageForError(l, error) ?? l.errorUnknown,
                onRetry: () => unawaited(
                  ref.read(sessionControllerProvider.notifier).retryRestore(),
                ),
              ),
      ),
    );
  }
}
