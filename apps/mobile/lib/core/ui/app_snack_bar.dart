import 'package:flutter/material.dart';

import 'package:siginak/core/theme/game_theme.dart';

/// Kısa bilgi / hata mesajı gösterir; önceki mesajı kapatır.
void showAppSnackBar(
  BuildContext context,
  String message, {
  bool isError = false,
  String? actionLabel,
  VoidCallback? onAction,
}) {
  final game = context.gameTheme;
  final messenger = ScaffoldMessenger.of(context)..hideCurrentSnackBar();
  messenger.showSnackBar(
    SnackBar(
      content: Row(
        children: [
          if (isError) ...[
            Icon(Icons.error_outline_rounded, color: game.danger, size: 20),
            const SizedBox(width: 8),
          ],
          Expanded(child: Text(message)),
        ],
      ),
      action: actionLabel != null && onAction != null
          ? SnackBarAction(label: actionLabel, onPressed: onAction)
          : null,
    ),
  );
}
