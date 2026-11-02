import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/config/app_config_provider.dart';

/// Ana ekran (geçici içerik). Gerçek içerik UI-004 mockup'ına göre
/// ROOM-011 / PLAY görevlerinde oluşturulacaktır.
class HomeScreen extends ConsumerWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final config = ref.watch(appConfigProvider);
    final textTheme = Theme.of(context).textTheme;
    return Scaffold(
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('SIĞINAK', style: textTheme.displayMedium),
            const SizedBox(height: 8),
            Text('Son Kalanlar', style: textTheme.titleMedium),
            if (!config.isProduction) ...[
              const SizedBox(height: 24),
              Text(
                'Ortam: ${config.environment.name} · ${config.apiBaseUrl}',
                style: textTheme.bodySmall,
              ),
            ],
          ],
        ),
      ),
    );
  }
}
