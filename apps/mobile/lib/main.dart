import 'package:flutter/material.dart';

import 'core/config/app_config.dart';

void main() {
  final config = AppConfig.fromEnvironment();
  runApp(SiginakApp(config: config));
}

/// Uygulamanın kök widget'ı.
///
/// Tema (MOB-008), yönlendirme (MOB-005) ve durum yönetimi (MOB-004)
/// sonraki görevlerde eklenecektir.
class SiginakApp extends StatelessWidget {
  const SiginakApp({required this.config, super.key});

  final AppConfig config;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Sığınak',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        useMaterial3: true,
        brightness: Brightness.dark,
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFFF2A33A),
          brightness: Brightness.dark,
        ),
        scaffoldBackgroundColor: const Color(0xFF0F1115),
      ),
      home: _PlaceholderScreen(config: config),
    );
  }
}

class _PlaceholderScreen extends StatelessWidget {
  const _PlaceholderScreen({required this.config});

  final AppConfig config;

  @override
  Widget build(BuildContext context) {
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
