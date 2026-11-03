import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/app/router/app_router.dart';
import 'package:siginak/core/theme/app_theme.dart';

/// Uygulamanın kök widget'ı: tema ve yönlendirmenin bağlandığı yer.
///
/// Bir `ProviderScope` altında çalışır (bkz. `main.dart`).
/// MVP'de yalnızca koyu tema vardır (docs/ui/brand.md §2).
class SiginakApp extends ConsumerWidget {
  const SiginakApp({super.key});

  static final _theme = buildAppTheme();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(
      title: 'Sığınak',
      debugShowCheckedModeBanner: false,
      theme: _theme,
      darkTheme: _theme,
      themeMode: ThemeMode.dark,
      routerConfig: ref.watch(appRouterProvider),
    );
  }
}
