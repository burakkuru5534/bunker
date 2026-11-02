import 'package:flutter/material.dart';

import 'package:siginak/core/ui/placeholder_screen.dart';

/// `/join` — davet koduyla odaya katılma (UI mockup: "Koda katıl").
class JoinScreen extends StatelessWidget {
  const JoinScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const PlaceholderScreen(title: 'Koda katıl', task: 'ROOM-011');
  }
}
