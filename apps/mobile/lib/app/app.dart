import 'package:flutter/material.dart';

import 'package:siginak/core/config/app_config.dart';
import 'package:siginak/features/home/presentation/home_screen.dart';

/// Uygulamanın kök widget'ı: tema, yönlendirme ve bağımlılıkların bağlandığı yer.
///
/// Tema (MOB-008), yönlendirme (MOB-005) ve durum yönetimi (MOB-004)
/// sonraki görevlerde buraya eklenecektir.
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
      home: HomeScreen(config: config),
    );
  }
}
