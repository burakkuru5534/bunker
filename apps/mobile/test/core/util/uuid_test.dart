import 'dart:math';

import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/util/uuid.dart';

void main() {
  final v4 = RegExp(
    r'^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$',
  );

  test('UUID v4 biçimi ve benzersizlik', () {
    final ids = {for (var i = 0; i < 1000; i++) uuidV4()};
    expect(ids, hasLength(1000));
    expect(ids.every(v4.hasMatch), isTrue);
  });

  test('sabit kaynakla sürüm ve varyant bitleri', () {
    expect(uuidV4(_Max()), 'ffffffff-ffff-4fff-bfff-ffffffffffff');
    expect(uuidV4(_Zero()), '00000000-0000-4000-8000-000000000000');
  });
}

class _Max implements Random {
  @override
  int nextInt(int max) => max - 1;
  @override
  bool nextBool() => true;
  @override
  double nextDouble() => 0.99;
}

class _Zero implements Random {
  @override
  int nextInt(int max) => 0;
  @override
  bool nextBool() => false;
  @override
  double nextDouble() => 0;
}
