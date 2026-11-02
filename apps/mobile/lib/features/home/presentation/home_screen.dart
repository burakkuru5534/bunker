import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/config/app_config_provider.dart';

/// `/home` — ana ekran (geçici içerik). Gerçek içerik UI-004 mockup'ına
/// göre ROOM-011 görevinde oluşturulacaktır.
///
/// Gezinme geri çağrılarla verilir; ekran rota yollarını bilmez.
class HomeScreen extends ConsumerWidget {
  const HomeScreen({
    required this.onJoin,
    required this.onOpenSettings,
    super.key,
  });

  final VoidCallback onJoin;
  final VoidCallback onOpenSettings;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final config = ref.watch(appConfigProvider);
    final textTheme = Theme.of(context).textTheme;
    return Scaffold(
      appBar: AppBar(
        actions: [
          IconButton(
            tooltip: 'Ayarlar',
            onPressed: onOpenSettings,
            icon: const Icon(Icons.settings_outlined),
          ),
        ],
      ),
      body: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text('SIĞINAK', style: textTheme.displayMedium),
            const SizedBox(height: 8),
            Text('Son Kalanlar', style: textTheme.titleMedium),
            const SizedBox(height: 32),
            OutlinedButton(onPressed: onJoin, child: const Text('Koda katıl')),
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
