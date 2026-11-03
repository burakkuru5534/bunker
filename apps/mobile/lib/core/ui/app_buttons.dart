import 'package:flutter/material.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/theme/game_theme.dart';

enum _AppButtonKind { primary, secondary, danger }

/// Ortak düğme (docs/ui/design-system.md §3).
///
/// - `onPressed == null` → devre dışı; `disabledReason` altında gösterilir.
/// - `isLoading` → spinner gösterir ve tekrar basılamaz.
class AppButton extends StatelessWidget {
  const AppButton.primary({
    required this.label,
    required this.onPressed,
    this.icon,
    this.isLoading = false,
    this.disabledReason,
    this.expand = true,
    super.key,
  }) : _kind = _AppButtonKind.primary;

  const AppButton.secondary({
    required this.label,
    required this.onPressed,
    this.icon,
    this.isLoading = false,
    this.disabledReason,
    this.expand = true,
    super.key,
  }) : _kind = _AppButtonKind.secondary;

  /// Tehlikeli eylem (ayrıl, hesabı sil, oy ver).
  const AppButton.danger({
    required this.label,
    required this.onPressed,
    this.icon,
    this.isLoading = false,
    this.disabledReason,
    this.expand = true,
    super.key,
  }) : _kind = _AppButtonKind.danger;

  final String label;
  final VoidCallback? onPressed;
  final IconData? icon;
  final bool isLoading;
  final String? disabledReason;
  final bool expand;
  final _AppButtonKind _kind;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final game = context.gameTheme;
    final enabled = onPressed != null && !isLoading;
    final action = enabled ? onPressed : null;

    final Color spinnerColor;
    final ButtonStyleButton button;
    final child = isLoading
        ? null
        : Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              if (icon != null) ...[
                Icon(icon, size: 20),
                const SizedBox(width: AppSpacing.s2),
              ],
              Flexible(child: Text(label, overflow: TextOverflow.ellipsis)),
            ],
          );

    switch (_kind) {
      case _AppButtonKind.primary:
        spinnerColor = theme.colorScheme.onPrimary;
        button = FilledButton(
          onPressed: action,
          child: child ?? _Spinner(color: spinnerColor),
        );
      case _AppButtonKind.secondary:
        spinnerColor = theme.colorScheme.onSurface;
        button = OutlinedButton(
          onPressed: action,
          child: child ?? _Spinner(color: spinnerColor),
        );
      case _AppButtonKind.danger:
        spinnerColor = game.danger;
        button = OutlinedButton(
          onPressed: action,
          style: OutlinedButton.styleFrom(
            foregroundColor: game.danger,
            backgroundColor: Colors.transparent,
            side: BorderSide(color: game.danger),
          ),
          child: child ?? _Spinner(color: spinnerColor),
        );
    }

    final semantic = Semantics(
      button: true,
      enabled: enabled,
      label: isLoading ? context.l10n.loadingLabel(label) : null,
      excludeSemantics: isLoading,
      child: expand ? SizedBox(width: double.infinity, child: button) : button,
    );

    final reason = disabledReason;
    if (onPressed != null || reason == null) return semantic;
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        semantic,
        const SizedBox(height: AppSpacing.s2),
        Text(
          reason,
          textAlign: TextAlign.center,
          style: theme.textTheme.bodySmall,
        ),
      ],
    );
  }
}

class _Spinner extends StatelessWidget {
  const _Spinner({required this.color});

  final Color color;

  @override
  Widget build(BuildContext context) {
    return SizedBox.square(
      dimension: 20,
      child: CircularProgressIndicator(strokeWidth: 2.5, color: color),
    );
  }
}
