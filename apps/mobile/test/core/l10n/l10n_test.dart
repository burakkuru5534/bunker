import 'dart:convert';
import 'dart:io';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/l10n/l10n.dart';

void main() {
  test('ARB dosyasında boş metin yok', () {
    final arb = jsonDecode(
      File('lib/core/l10n/arb/app_tr.arb').readAsStringSync(),
    ) as Map<String, Object?>;

    for (final entry in arb.entries) {
      if (entry.key.startsWith('@')) continue;
      expect(entry.value, isA<String>(), reason: entry.key);
      expect((entry.value! as String).trim(), isNotEmpty, reason: entry.key);
    }
  });

  test('yalnızca Türkçe desteklenir ve yer tutucular çalışır', () {
    expect(AppLocalizations.supportedLocales, [const Locale('tr')]);
    final l = lookupAppLocalizations(const Locale('tr'));

    expect(l.loadingLabel('Katıl'), 'Katıl, yükleniyor');
    expect(l.avatarEliminated('Deniz'), 'Deniz, elendi');
    expect(l.inviteCodeLabel(6), 'Davet kodu, 6 karakter');
    expect(l.errorRateLimited(3), contains('3 saniye'));
  });
}
