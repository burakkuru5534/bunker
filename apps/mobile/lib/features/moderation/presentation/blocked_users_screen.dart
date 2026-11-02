import 'package:flutter/material.dart';

import 'package:siginak/core/ui/placeholder_screen.dart';

/// `/settings/blocks` — engellenen oyuncular listesi.
class BlockedUsersScreen extends StatelessWidget {
  const BlockedUsersScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const PlaceholderScreen(
      title: 'Engellenen oyuncular',
      task: 'SAFE-006',
    );
  }
}
