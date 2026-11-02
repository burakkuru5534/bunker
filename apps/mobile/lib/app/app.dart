import 'package:flutter/material.dart';

import 'package:siginak/features/home/presentation/home_screen.dart';

/// Uygulamanın kök widget'ı: tema, yönlendirme ve bağımlılıkların bağlandığı yer.
///
/// Bir `ProviderScope` altında çalışır (bkz. `main.dart`).
/// Tema (MOB-008) ve yönlendirme (MOB-005) sonraki görevlerde eklenecektir.
class SiginakApp extends StatelessWidget {
  const SiginakApp({super.key});

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
      home: const HomeScreen(),
    );
  }
}
