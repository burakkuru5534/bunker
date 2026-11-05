import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/ui/app_snack_bar.dart';
import 'package:siginak/core/ui/confirm_dialog.dart';
import 'package:siginak/core/ui/state_views.dart';
import 'package:siginak/features/startup/presentation/update_required_screen.dart';

import '../../helpers/pump_themed.dart';
import '../../helpers/small_screen.dart';

void main() {
  Future<bool?> openDialog(WidgetTester t, String tapLabel) async {
    bool? result;
    await pumpThemed(
      t,
      Builder(
        builder: (context) => TextButton(
          onPressed: () async => result = await showConfirmDialog(
            context,
            title: 'Odadan ayrıl?',
            message: 'Ayrılırsan elenmiş sayılırsın.',
            confirmLabel: 'Ayrıl',
            destructive: true,
          ),
          child: const Text('aç'),
        ),
      ),
    );
    await t.tap(find.text('aç'));
    await t.pumpAndSettle();
    expect(find.text('Odadan ayrıl?'), findsOneWidget);
    await t.tap(find.text(tapLabel));
    await t.pumpAndSettle();
    return result;
  }

  testWidgets('onay penceresi: onayla → true', (t) async {
    expect(await openDialog(t, 'Ayrıl'), isTrue);
  });

  testWidgets('onay penceresi: vazgeç → false', (t) async {
    expect(await openDialog(t, 'Vazgeç'), isFalse);
  });

  testWidgets('snackbar mesajı ve hata ikonu', (t) async {
    await pumpThemed(
      t,
      Builder(
        builder: (context) => TextButton(
          onPressed: () =>
              showAppSnackBar(context, 'Bu oda dolu.', isError: true),
          child: const Text('göster'),
        ),
      ),
    );
    await t.tap(find.text('göster'));
    await t.pump();

    expect(find.text('Bu oda dolu.'), findsOneWidget);
    expect(find.byIcon(Icons.error_outline_rounded), findsOneWidget);
  });

  testWidgets('yükleniyor, hata ve boş görünümleri', (t) async {
    await pumpThemed(t, const LoadingView(message: 'Bağlanılıyor'));
    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(find.text('Bağlanılıyor'), findsOneWidget);

    var retries = 0;
    await pumpThemed(
      t,
      ErrorView(message: 'Bir şeyler ters gitti.', onRetry: () => retries++),
    );
    await t.tap(find.text('Tekrar dene'));
    expect(retries, 1);

    await pumpThemed(
      t,
      const EmptyState(icon: Icons.block, message: 'Engellediğin kimse yok.'),
    );
    expect(find.text('Engellediğin kimse yok.'), findsOneWidget);
  });

  testWidgets('320 dp ve %200 yazı: durum görünümleri ve onay penceresi', (
    t,
  ) async {
    useSmallScreen(t);
    for (final w in <Widget>[
      LoadingView(message: 'Sonuçlar yükleniyor, lütfen bekle. ' * 6),
      ErrorView(
        message: 'Oda yüklenemedi. Tekrar dene; sorun sürerse odadan çıkıp yeniden katıl.',
        onRetry: () {},
      ),
      const EmptyState(
        icon: Icons.block_rounded,
        message: 'Engellediğin kimse yok.',
      ),
    ]) {
      await pumpThemed(t, w);
      expect(t.takeException(), isNull, reason: '$w');
    }
    await openDialog(t, 'Vazgeç');
    expect(t.takeException(), isNull);
  });

  testWidgets('320 dp ve %200 yazı: güncelleme ekranı', (t) async {
    useSmallScreen(t);
    await t.pumpWidget(
      const MaterialApp(
        locale: Locale('tr'),
        supportedLocales: AppLocalizations.supportedLocales,
        localizationsDelegates: testLocalizationsDelegates,
        home: UpdateRequiredScreen(),
      ),
    );
    expect(t.takeException(), isNull);
  });
}
