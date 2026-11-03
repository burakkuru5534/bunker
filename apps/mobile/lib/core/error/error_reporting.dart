import 'dart:ui';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';

import 'package:siginak/core/theme/app_colors.dart';

/// Yakalanmamış hataları raporlayan fonksiyon.
typedef ErrorReporter = void Function(Object error, StackTrace? stack);

/// Varsayılan raporlayıcı: yalnızca debug derlemede konsola yazar.
///
/// Crash raporlama servisi REL-004'te bağlanacaktır. Hata mesajları kullanıcı
/// verisi (token, sohbet içeriği) içermemelidir; yalnızca tür ve yığın izi
/// raporlanır.
void defaultErrorReporter(Object error, StackTrace? stack) {
  if (kDebugMode) {
    debugPrint('[hata] ${error.runtimeType}: $error');
    if (stack != null) debugPrintStack(stackTrace: stack, maxFrames: 8);
  }
}

/// Uygulama genelinde yakalanmamış hata işleyicilerini kurar.
///
/// - Çerçeve hataları (`FlutterError.onError`): debug'da kırmızı ekran ve
///   konsol; her durumda raporlanır.
/// - Asenkron / platform hataları (`PlatformDispatcher.onError`): raporlanır,
///   uygulama çökmez.
/// - Release derlemede bozuk widget'lar kırmızı ekran yerine sade bir
///   kutu gösterir.
void installGlobalErrorHandlers({
  ErrorReporter reporter = defaultErrorReporter,
  required String fallbackMessage,
}) {
  FlutterError.onError = (details) {
    if (kDebugMode) FlutterError.presentError(details);
    reporter(details.exception, details.stack);
  };

  PlatformDispatcher.instance.onError = (error, stack) {
    reporter(error, stack);
    return true;
  };

  if (kReleaseMode) {
    ErrorWidget.builder = (details) =>
        ReleaseErrorBox(message: fallbackMessage);
  }
}

/// Release derlemede bozulan bir widget'ın yerine gösterilen kutu.
class ReleaseErrorBox extends StatelessWidget {
  const ReleaseErrorBox({required this.message, super.key});

  final String message;

  @override
  Widget build(BuildContext context) {
    return Container(
      alignment: Alignment.center,
      padding: const EdgeInsets.all(16),
      color: AppColors.surface,
      child: Text(
        message,
        textAlign: TextAlign.center,
        textDirection: TextDirection.ltr,
        style: const TextStyle(color: AppColors.textSecondary, fontSize: 14),
      ),
    );
  }
}
