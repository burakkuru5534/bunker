import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/app/router/app_router.dart';
import 'package:siginak/core/l10n/l10n.dart';
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
      onGenerateTitle: (context) => context.l10n.appTitle,
      locale: const Locale('tr'),
      supportedLocales: AppLocalizations.supportedLocales,
      localizationsDelegates: const [
        AppLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      debugShowCheckedModeBanner: false,
      theme: _theme,
      darkTheme: _theme,
      themeMode: ThemeMode.dark,
      routerConfig: ref.watch(appRouterProvider),
    );
  }
}
