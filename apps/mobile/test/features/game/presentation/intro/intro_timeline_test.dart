import 'dart:io';
import 'dart:ui' as ui;

import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/l10n/generated/app_localizations_tr.dart';
import 'package:siginak/features/game/presentation/intro/intro_timeline.dart';
import 'package:siginak/features/game/presentation/intro/scenario_cinematic.dart';

void main() {
  final l = AppLocalizationsTr();
  // Katalogdaki senaryolar (migrations/postgres/0007_catalog_content.sql).
  const catalog = [
    'nuclear_winter',
    'red_cough',
    'meteor',
    'silent_machines',
    'solar_storm',
  ];

  IntroTimeline timeline(String id) =>
      IntroTimeline(introScripts[id]!, introNarration(l, id)!);

  test('her katalog senaryosunun tarifi, metni ve dosyaları var', () {
    expect(introScripts.keys, unorderedEquals(catalog));
    for (final id in catalog) {
      final text = introNarration(l, id);
      expect(text, isNotNull, reason: id);
      expect(text!.split('\n'), hasLength(4), reason: '$id: dört bölüm');
      for (final f in [
        'back.webp',
        'mid.webp',
        'front.webp',
        'hero.webp',
        'narration.ogg',
        'impact.ogg',
      ]) {
        expect(
          File('assets/scenarios/$id/$f').existsSync(),
          isTrue,
          reason: '$id/$f',
        );
      }
    }
    for (final f in ['door.webp', 'music.ogg']) {
      expect(File('assets/scenarios/$f').existsSync(), isTrue, reason: f);
    }
    expect(introNarration(l, 'yeni_senaryo'), isNull);
  });

  testWidgets('görseller çözülebilir (WebP, beklenen boyutlar)', (
    tester,
  ) async {
    await tester.runAsync(() async {
      final files = [
        for (final id in catalog)
          for (final n in ['back', 'mid', 'front', 'hero'])
            'assets/scenarios/$id/$n.webp',
        'assets/scenarios/door.webp',
      ];
      for (final f in files) {
        final codec = await ui.instantiateImageCodec(File(f).readAsBytesSync());
        final frame = await codec.getNextFrame();
        expect(frame.image.width, greaterThanOrEqualTo(700), reason: f);
        frame.image.dispose();
      }
    });
  });

  test('pubspec senaryo klasörlerini varlık olarak listeler', () {
    final pubspec = File('pubspec.yaml').readAsStringSync();
    expect(pubspec, contains('- assets/scenarios/\n'));
    for (final id in catalog) {
      expect(pubspec, contains('- assets/scenarios/$id/\n'), reason: id);
    }
  });

  for (final id in catalog) {
    test('$id: olaylar sıralı, anlatım kapanıştan önce biter', () {
      final tl = timeline(id);
      expect(tl.impactAt, greaterThan(kNarrationStart + 1));
      expect(tl.shelterAt, greaterThan(tl.impactAt + 3));
      expect(tl.doorAt, greaterThan(tl.shelterAt + 2));
      expect(tl.narrationEnd, lessThanOrEqualTo(IntroTimeline.length - 4));

      // Altyazılar boşluksuz, sıralı ve anlatımı tam kaplar.
      final cues = tl.cues;
      expect(cues.first.start, kNarrationStart);
      expect(cues.last.end, closeTo(tl.narrationEnd, 1e-9));
      expect(cues.last.text, endsWith('Kim içeri girecek?'));
      for (var i = 1; i < cues.length; i++) {
        expect(cues[i].start, closeTo(cues[i - 1].end, 1e-9));
        expect(cues[i].end, greaterThan(cues[i].start));
      }
      expect(tl.cueAt(1), isNull); // başlık anı
      expect(tl.cueAt(kNarrationStart + 0.01), cues.first.text);
      expect(tl.cueAt(tl.narrationEnd + 0.1), isNull);
    });
  }

  test('anlatım bölümleri görsel olaylara bağlı', () {
    final tl = timeline('meteor');
    expect(tl.cueAt(tl.impactAt + 0.05), 'Dev bir göktaşı okyanusa çarptı.');
    expect(
      tl.cueAt(tl.shelterAt + 0.05),
      'Sığınakta temiz su ve koca bir kütüphane var.',
    );
    expect(tl.cueAt(tl.doorAt + 0.05), 'Herkese yer yok.');
  });

  test('katmanlar: başlık, kapı, kararma', () {
    final tl = timeline('nuclear_winter');
    expect(tl.titleOpacity(1.5), 1);
    expect(tl.titleOpacity(3), 0);
    expect(tl.backOpacity(0.5), 0);
    expect(tl.backOpacity(3), 1);
    expect(tl.frontOpacity(tl.shelterAt - 0.1), 0);
    expect(tl.frontOpacity(tl.shelterAt + 1.5), 1);
    expect(tl.frontShift(tl.shelterAt + 3), 0);
    expect(tl.doorOpacity(tl.doorAt - 0.1), 0);
    expect(tl.doorOpacity(tl.doorAt + 1.5), 1);
    expect(tl.fadeOut(26), 0);
    expect(tl.fadeOut(29.5), 1);
  });

  test('çarpma: parlama ve sarsıntı; hareket azaltılmışsa yok', () {
    final tl = timeline('meteor');
    final at = tl.impactAt + 0.1;
    expect(tl.flash(at, reduceMotion: false), greaterThan(0.5));
    expect(tl.shake(at, reduceMotion: false).distance, greaterThan(0));
    expect(tl.flash(at, reduceMotion: true), 0);
    expect(tl.shake(at, reduceMotion: true), Offset.zero);
    expect(tl.flash(tl.impactAt - 0.1, reduceMotion: false), 0);
    expect(tl.flash(tl.impactAt + 2, reduceMotion: false), 0);
  });

  test('ana nesne hareketleri', () {
    final meteor = timeline('meteor');
    expect(meteor.hero(meteor.impactAt - 3).opacity, 0);
    expect(meteor.hero(meteor.impactAt - 1).opacity, 1);
    expect(meteor.hero(meteor.impactAt + 0.5).opacity, 0); // çarptı

    final drone = timeline('silent_machines');
    expect(drone.hero(drone.impactAt + 1).center.dx, closeTo(0.5, 1e-9));
    expect(drone.hero(drone.shelterAt + 2).center.dx, greaterThan(1));

    final sun = timeline('solar_storm');
    expect(
      sun.hero(sun.impactAt).center.dy,
      lessThan(sun.hero(sun.impactAt - 2.5).center.dy),
    ); // yükselir

    final flash = timeline('nuclear_winter');
    expect(flash.hero(flash.impactAt - 1).opacity, 0);
    expect(flash.hero(flash.impactAt + 2).scale, 1);
  });

  test('süre sunucudaki SCENARIO_INTRO süresiyle aynı', () {
    final go = File('../../backend/internal/game/engine/durations.go');
    if (!go.existsSync()) return; // yalnızca mobil klasörü varsa
    final m = RegExp(r'ScenarioIntro:\s+(\d+) \* time\.Second')
        .firstMatch(go.readAsStringSync());
    expect(m, isNotNull);
    expect(int.parse(m!.group(1)!), kIntroLength.inSeconds);
  });

  test('başlangıç anı kalan süreden hesaplanır', () {
    expect(introStartAt(null), 0);
    expect(introStartAt(kIntroLength), 0);
    expect(introStartAt(const Duration(seconds: 45)), 0); // saat farkı
    expect(introStartAt(const Duration(seconds: 10)), 20);
    expect(introStartAt(const Duration(milliseconds: 29500)), 0.5);
    expect(introStartAt(Duration.zero), 30);
  });
}
