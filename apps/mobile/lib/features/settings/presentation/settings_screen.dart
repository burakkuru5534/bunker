import 'package:flutter/material.dart';

import 'package:siginak/core/ui/placeholder_screen.dart';

/// `/settings` — takma ad, engellenenler, gizlilik, hesap silme.
class SettingsScreen extends StatelessWidget {
  const SettingsScreen({required this.onOpenBlockedUsers, super.key});

  final VoidCallback onOpenBlockedUsers;

  @override
  Widget build(BuildContext context) {
    return PlaceholderScreen(
      title: 'Ayarlar',
      task: 'SAFE-006 / SAFE-009',
      children: [
        OutlinedButton(
          onPressed: onOpenBlockedUsers,
          child: const Text('Engellenen oyuncular'),
        ),
      ],
    );
  }
}
