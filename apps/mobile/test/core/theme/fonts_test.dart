import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/theme/font_licenses.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const fontFiles = [
    'assets/fonts/oswald/Oswald-Medium.ttf',
    'assets/fonts/oswald/Oswald-SemiBold.ttf',
    'assets/fonts/oswald/Oswald-Bold.ttf',
    'assets/fonts/inter/Inter-Regular.ttf',
    'assets/fonts/inter/Inter-Medium.ttf',
    'assets/fonts/inter/Inter-SemiBold.ttf',
  ];

  test('yazı tipi dosyaları var, TrueType ve pubspec\'te kayıtlı', () {
    final pubspec = File('pubspec.yaml').readAsStringSync();
    for (final path in fontFiles) {
      final bytes = File(path).readAsBytesSync();
      final magic = ByteData.sublistView(bytes, 0, 4).getUint32(0);
      expect(magic, 0x00010000, reason: '$path TrueType değil');
      expect(pubspec, contains(path));
    }
  });

  test('OFL lisansları lisans sayfasına eklenir', () async {
    registerFontLicenses();
    final packages = <String>{};
    await for (final entry in LicenseRegistry.licenses) {
      packages.addAll(entry.packages);
    }

    expect(packages, containsAll(['Oswald', 'Inter']));
  });
}
