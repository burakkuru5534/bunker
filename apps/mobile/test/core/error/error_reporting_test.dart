import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/error/error_reporting.dart';

import '../../helpers/pump_themed.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('çerçeve ve platform hataları raporlanır, uygulama çökmez', () {
    final originalFlutter = FlutterError.onError;
    final originalPlatform = PlatformDispatcher.instance.onError;
    addTearDown(() {
      FlutterError.onError = originalFlutter;
      PlatformDispatcher.instance.onError = originalPlatform;
    });

    final reported = <Object>[];
    installGlobalErrorHandlers(
      reporter: (e, _) => reported.add(e),
      fallbackMessage: 'gösterilemedi',
    );

    final frameworkError = StateError('widget');
    final asyncError = StateError('async');
    // Debug'da presentError konsola yazar; testte yalnızca raporlamayı ölçeriz.
    FlutterError.onError!(
      FlutterErrorDetails(exception: frameworkError, silent: true),
    );
    final handled = PlatformDispatcher.instance.onError!(
      asyncError,
      StackTrace.current,
    );

    expect(reported, [frameworkError, asyncError]);
    expect(handled, isTrue);
  });

  testWidgets('release hata kutusu mesajı gösterir', (t) async {
    await pumpThemed(
      t,
      const ReleaseErrorBox(message: 'Bu bölüm gösterilemedi.'),
    );

    expect(find.text('Bu bölüm gösterilemedi.'), findsOneWidget);
  });
}
