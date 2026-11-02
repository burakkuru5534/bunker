import 'package:flutter/material.dart';

void main() {
  runApp(const SiginakApp());
}

/// Uygulamanın kök widget'ı.
///
/// MOB-001 kapsamında yalnızca derlemenin doğrulanması için sade bir
/// başlangıç ekranı gösterir. Tema (MOB-008), yönlendirme (MOB-005) ve
/// durum yönetimi (MOB-004) sonraki görevlerde eklenecektir.
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
      home: const _PlaceholderScreen(),
    );
  }
}

class _PlaceholderScreen extends StatelessWidget {
  const _PlaceholderScreen();

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
          ],
        ),
      ),
    );
  }
}
