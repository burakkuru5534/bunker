import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/ui/invite_code_input.dart';

import '../../helpers/pump_themed.dart';

void main() {
  group('InviteCodeFormatter.normalize', () {
    test('büyük harfe çevirir ve 6 ile sınırlar', () {
      expect(InviteCodeFormatter.normalize('k7m2qx'), 'K7M2QX');
      expect(InviteCodeFormatter.normalize('K7M2QXABC'), 'K7M2QX');
    });

    test('belirsiz ve alfabe dışı karakterleri atar', () {
      expect(InviteCodeFormatter.normalize('0O1IL'), '');
      expect(InviteCodeFormatter.normalize('k7-m2 qx'), 'K7M2QX');
      expect(InviteCodeFormatter.normalize('ğüş'), '');
    });
  });

  testWidgets('yazılan kod kutulara dağılır, tamamlanınca bildirilir', (
    t,
  ) async {
    String? completed;
    final changes = <String>[];
    await pumpThemed(
      t,
      InviteCodeInput(
        onChanged: changes.add,
        onCompleted: (c) => completed = c,
      ),
    );

    await t.enterText(find.byType(TextField), 'k7m');
    await t.pump();
    expect(find.text('K'), findsOneWidget);
    expect(find.text('7'), findsOneWidget);
    expect(completed, isNull);

    await t.enterText(find.byType(TextField), 'k7m2-qx');
    await t.pump();
    expect(completed, 'K7M2QX');
    expect(changes.last, 'K7M2QX');
  });

  testWidgets('erişilebilirlik etiketi ve 6 kutu', (t) async {
    await pumpThemed(t, const InviteCodeInput());

    expect(find.bySemanticsLabel(RegExp('Davet kodu')), findsOneWidget);
    for (var i = 0; i < inviteCodeLength; i++) {
      expect(find.byKey(ValueKey('invite-box-$i')), findsOneWidget);
    }
  });
}
