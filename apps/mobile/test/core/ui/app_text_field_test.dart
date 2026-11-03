import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/ui/app_text_field.dart';

import '../../helpers/pump_themed.dart';

void main() {
  testWidgets('etiket, sayaç ve uzunluk sınırı', (t) async {
    final controller = TextEditingController();
    addTearDown(controller.dispose);
    await pumpThemed(
      t,
      AppTextField(
        label: 'Takma adın',
        helperText: '3–16 karakter',
        maxLength: 16,
        controller: controller,
      ),
    );

    await t.enterText(find.byType(TextField), 'Burak');
    await t.pump();
    expect(find.text('Takma adın'), findsOneWidget);
    expect(find.text('5/16'), findsOneWidget);

    await t.enterText(find.byType(TextField), 'a' * 30);
    expect(controller.text.length, 16);
  });

  testWidgets('hata metni gösterilir', (t) async {
    await pumpThemed(
      t,
      const AppTextField(label: 'Takma adın', errorText: 'En az 3 karakter'),
    );

    expect(find.text('En az 3 karakter'), findsOneWidget);
  });
}
