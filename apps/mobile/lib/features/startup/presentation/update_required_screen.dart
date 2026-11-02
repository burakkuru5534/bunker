import 'package:flutter/material.dart';

import 'package:siginak/core/ui/placeholder_screen.dart';

/// `/update` — uygulama sürümü artık desteklenmiyor.
class UpdateRequiredScreen extends StatelessWidget {
  const UpdateRequiredScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const PlaceholderScreen(
      title: 'Yeni bir sürüm var',
      task: 'MOB-009',
      showAppBar: false,
    );
  }
}
