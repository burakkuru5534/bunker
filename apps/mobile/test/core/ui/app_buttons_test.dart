import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/ui/app_buttons.dart';

import '../../helpers/pump_themed.dart';

void main() {
  testWidgets('basınca geri çağrı çalışır; en az 48 dp yükseklik', (t) async {
    var taps = 0;
    await pumpThemed(
      t,
      AppButton.primary(label: 'Başla', onPressed: () => taps++),
    );

    await t.tap(find.text('Başla'));
    expect(taps, 1);
    expect(
      t.getSize(find.byType(FilledButton)).height,
      greaterThanOrEqualTo(48),
    );
  });

  testWidgets('devre dışıyken neden gösterilir ve basılamaz', (t) async {
    await pumpThemed(
      t,
      const AppButton.primary(
        label: 'Oyunu başlat',
        onPressed: null,
        disabledReason: '2 kişi bekleniyor',
      ),
    );

    expect(find.text('2 kişi bekleniyor'), findsOneWidget);
    final button = t.widget<FilledButton>(find.byType(FilledButton));
    expect(button.onPressed, isNull);
  });

  testWidgets('yüklenirken spinner gösterir ve tekrar basılamaz', (t) async {
    var taps = 0;
    await pumpThemed(
      t,
      AppButton.secondary(
        label: 'Katıl',
        onPressed: () => taps++,
        isLoading: true,
      ),
    );

    expect(find.byType(CircularProgressIndicator), findsOneWidget);
    expect(find.text('Katıl'), findsNothing);
    await t.tap(find.byType(OutlinedButton));
    expect(taps, 0);
    expect(find.bySemanticsLabel('Katıl, yükleniyor'), findsOneWidget);
  });

  testWidgets('ikonlu tehlike düğmesi', (t) async {
    await pumpThemed(
      t,
      AppButton.danger(
        label: 'Hesabımı sil',
        icon: Icons.delete_outline,
        onPressed: () {},
      ),
    );

    expect(find.byIcon(Icons.delete_outline), findsOneWidget);
    expect(find.byType(OutlinedButton), findsOneWidget);
  });
}
