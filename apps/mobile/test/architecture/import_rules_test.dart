import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

/// Katman ve feature sınırlarını otomatik denetler (ARCHITECTURE.md §2).
///
/// Kurallar:
/// 1. `core/` hiçbir zaman `app/` veya `features/` içe aktaramaz.
/// 2. Bir feature başka bir feature'ın içini içe aktaramaz; yalnızca
///    o feature'ın kökündeki ortak dosyayı (`features/<ad>/<ad>.dart`)
///    kullanabilir.
/// 3. `features/` içinden `app/` içe aktarılamaz.
void main() {
  final libDir = Directory('lib');
  final importPattern = RegExp(
    r'''^import\s+['"]package:siginak/([^'"]+)['"]''',
    multiLine: true,
  );

  List<(String file, String target)> imports() {
    final result = <(String, String)>[];
    for (final entity in libDir.listSync(recursive: true)) {
      if (entity is! File || !entity.path.endsWith('.dart')) continue;
      final rel = entity.path.replaceAll(r'\', '/').replaceFirst('lib/', '');
      for (final m in importPattern.allMatches(entity.readAsStringSync())) {
        result.add((rel, m.group(1)!));
      }
    }
    return result;
  }

  String? featureOf(String path) {
    final parts = path.split('/');
    return parts.length > 1 && parts[0] == 'features' ? parts[1] : null;
  }

  test('core/, app/ veya features/ içe aktarmaz', () {
    final violations = imports()
        .where((i) => i.$1.startsWith('core/'))
        .where((i) => i.$2.startsWith('app/') || i.$2.startsWith('features/'))
        .map((i) => '${i.$1} → ${i.$2}')
        .toList();
    expect(violations, isEmpty);
  });

  test('features/, app/ içe aktarmaz', () {
    final violations = imports()
        .where((i) => i.$1.startsWith('features/') && i.$2.startsWith('app/'))
        .map((i) => '${i.$1} → ${i.$2}')
        .toList();
    expect(violations, isEmpty);
  });

  test('bir feature başka feature\'ın iç dosyalarını içe aktarmaz', () {
    final violations = <String>[];
    for (final (file, target) in imports()) {
      final from = featureOf(file);
      final to = featureOf(target);
      if (from == null || to == null || from == to) continue;
      final publicApi = 'features/$to/$to.dart';
      if (target != publicApi) violations.add('$file → $target');
    }
    expect(violations, isEmpty);
  });
}
