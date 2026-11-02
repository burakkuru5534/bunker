import 'package:flutter/material.dart';

/// Henüz tasarımı uygulanmamış ekranlar için geçici iskelet.
///
/// Her ekran, ilgili görevde mockup'a göre gerçek içerikle değiştirilir.
class PlaceholderScreen extends StatelessWidget {
  const PlaceholderScreen({
    required this.title,
    required this.task,
    this.children = const [],
    this.showAppBar = true,
    super.key,
  });

  final String title;
  final String task;
  final List<Widget> children;
  final bool showAppBar;

  @override
  Widget build(BuildContext context) {
    final textTheme = Theme.of(context).textTheme;
    return Scaffold(
      appBar: showAppBar ? AppBar(title: Text(title)) : null,
      body: SafeArea(
        child: Center(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (!showAppBar) Text(title, style: textTheme.headlineMedium),
                const SizedBox(height: 8),
                Text(
                  'Bu ekran $task görevinde tamamlanacak.',
                  style: textTheme.bodyMedium,
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 24),
                ...children,
              ],
            ),
          ),
        ),
      ),
    );
  }
}
