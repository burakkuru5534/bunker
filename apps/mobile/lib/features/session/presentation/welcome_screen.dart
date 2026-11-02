import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/ui/placeholder_screen.dart';
import 'package:siginak/features/session/presentation/session_controller.dart';

/// `/welcome` — takma ad ve misafir oturumu (UI mockup: "Hoş geldin").
class WelcomeScreen extends ConsumerWidget {
  const WelcomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return PlaceholderScreen(
      title: 'Hoş geldin',
      task: 'AUTH-002 / ROOM-011',
      showAppBar: false,
      children: [
        FilledButton(
          onPressed: () => ref
              .read(sessionControllerProvider.notifier)
              .signInAsGuestPlaceholder(),
          child: const Text('Başla (geçici)'),
        ),
      ],
    );
  }
}
