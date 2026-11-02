import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/app/router/app_router.dart';

/// Uygulamanın kök widget'ı: tema ve yönlendirmenin bağlandığı yer.
///
/// Bir `ProviderScope` altında çalışır (bkz. `main.dart`).
/// Tema token'ları MOB-008'de eklenecektir.
class SiginakApp extends ConsumerWidget {
  const SiginakApp({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(
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
      routerConfig: ref.watch(appRouterProvider),
    );
  }
}
