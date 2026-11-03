import 'package:flutter/material.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/ui/app_buttons.dart';

/// Bilinmeyen veya geçersiz bir adrese gidildiğinde gösterilir.
class NotFoundScreen extends StatelessWidget {
  const NotFoundScreen({required this.onGoHome, super.key});

  final VoidCallback onGoHome;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                context.l10n.notFoundMessage,
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 16),
              AppButton.primary(
                label: context.l10n.actionGoHome,
                onPressed: onGoHome,
                expand: false,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
