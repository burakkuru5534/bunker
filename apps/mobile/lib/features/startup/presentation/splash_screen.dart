import 'package:flutter/material.dart';

/// `/` — açılışta oturum kontrol edilirken kısa süre görünür.
class SplashScreen extends StatelessWidget {
  const SplashScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const Scaffold(body: Center(child: CircularProgressIndicator()));
  }
}
