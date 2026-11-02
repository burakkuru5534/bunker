import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/config/app_version.dart';

void main() {
  test('kAppVersion, pubspec.yaml sürümüyle aynı', () {
    final pubspec = File('pubspec.yaml').readAsStringSync();
    final match = RegExp(
      r'^version:\s*([^\s+]+)',
      multiLine: true,
    ).firstMatch(pubspec);

    expect(match, isNotNull);
    expect(kAppVersion, match!.group(1));
  });
}
