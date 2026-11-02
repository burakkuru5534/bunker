import 'package:flutter/material.dart';

/// Bilinmeyen veya geçersiz bir adrese gidildiğinde gösterilir.
class NotFoundScreen extends StatelessWidget {
  const NotFoundScreen({required this.onGoHome, super.key});

  final VoidCallback onGoHome;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                'Aradığın sayfa bulunamadı.',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: onGoHome,
                child: const Text('Ana ekrana dön'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
