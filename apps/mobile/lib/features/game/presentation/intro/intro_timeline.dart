import 'dart:math' as math;
import 'dart:ui' show Color, Offset;

/// Senaryo tanıtımı animasyonunun süresi. Sunucudaki `SCENARIO_INTRO`
/// süresiyle aynıdır (backend `engine.DefaultDurations.ScenarioIntro`).
const kIntroLength = Duration(seconds: 30);

/// Anlatımın başladığı an (sn). Öncesinde başlık görünür.
const kNarrationStart = 2.0;

/// Ana nesnenin hareket biçimi.
enum HeroMotion {
  /// Ufukta büyüyen parlama (patlama).
  flash,

  /// Sağ üstten çapraz düşüş, çarpmada kaybolur.
  fall,

  /// Yerinde belirir, ışığı titrer.
  flicker,

  /// Soldan girer, süzülür, sağdan çıkar.
  hover,

  /// Aşağıdan yükselir, nabız gibi atar.
  rise,
}

/// Parçacık türü (kodla çizilir, görsellerde yoktur).
enum ParticleKind { ash, fog, embers, scan, sparks }

/// Bir senaryonun animasyon tarifi. Görseller ve sesler
/// `assets/scenarios/<id>/` altındadır.
class IntroScript {
  const IntroScript({
    required this.id,
    required this.narration,
    required this.hero,
    required this.particles,
    required this.flash,
    this.heroCenter = const Offset(0.5, 0.32),
    this.heroWidth = 0.62,
    this.flashStrength = 0.85,
  });

  final String id;

  /// Anlatım sesinin süresi (sn; `narration.ogg`).
  final double narration;
  final HeroMotion hero;
  final ParticleKind particles;

  /// Çarpma anındaki parlamanın rengi ve en yüksek opaklığı.
  final Color flash;
  final double flashStrength;

  /// Ana nesnenin durduğu yer (ekran oranı) ve genişliği (ekran oranı).
  final Offset heroCenter;
  final double heroWidth;

  String asset(String name) => 'assets/scenarios/$id/$name';
}

/// Bilinen senaryolar. Sunucu bilinmeyen bir senaryo gönderirse animasyon
/// gösterilmez, metin tanıtımı görünür.
const introScripts = <String, IntroScript>{
  'nuclear_winter': IntroScript(
    id: 'nuclear_winter',
    narration: 21.0,
    hero: HeroMotion.flash,
    particles: ParticleKind.ash,
    flash: Color(0xFFFFF4E0),
    heroCenter: Offset(0.5, 0.36),
    heroWidth: 1.2,
  ),
  'red_cough': IntroScript(
    id: 'red_cough',
    narration: 23.3,
    hero: HeroMotion.flicker,
    particles: ParticleKind.fog,
    flash: Color(0xFFB3261E),
    flashStrength: 0.5,
    heroCenter: Offset(0.5, 0.42),
    heroWidth: 0.55,
  ),
  'meteor': IntroScript(
    id: 'meteor',
    narration: 21.1,
    hero: HeroMotion.fall,
    particles: ParticleKind.embers,
    flash: Color(0xFFFFD08A),
    heroCenter: Offset(0.42, 0.4),
  ),
  'silent_machines': IntroScript(
    id: 'silent_machines',
    narration: 22.9,
    hero: HeroMotion.hover,
    particles: ParticleKind.scan,
    flash: Color(0xFFFF3B3B),
    flashStrength: 0.45,
    heroCenter: Offset(0.5, 0.3),
    heroWidth: 0.7,
  ),
  'solar_storm': IntroScript(
    id: 'solar_storm',
    narration: 21.9,
    hero: HeroMotion.rise,
    particles: ParticleKind.sparks,
    flash: Color(0xFFE8FFE0),
    heroCenter: Offset(0.5, 0.28),
    heroWidth: 0.58,
  ),
};

/// Altyazı satırı (sn).
class IntroCue {
  const IntroCue(this.start, this.end, this.text);

  final double start;
  final double end;
  final String text;
}

/// Ana nesnenin bir andaki durumu.
class HeroPose {
  const HeroPose({
    required this.opacity,
    required this.center,
    required this.scale,
  });

  static const hidden = HeroPose(opacity: 0, center: Offset.zero, scale: 1);

  final double opacity;

  /// Ekran oranı.
  final Offset center;
  final double scale;
}

/// Zaman çizelgesi: her şey yalnızca `t`'nin (sn) fonksiyonudur; böylece
/// oyuna sonradan katılan oyuncu herhangi bir andan başlayabilir.
///
/// Anlatım metni satırlara (`\n`) bölünmüş dört bölümdür: kuruluş, felaket,
/// sığınak, kapı. Bölümlerin başlangıcı sesteki konuşma uzunluğuna oranla
/// hesaplanır; görsel olaylar (çarpma, sığınak, kapı) bu anlara bağlıdır.
class IntroTimeline {
  factory IntroTimeline(IntroScript script, String narrationText) {
    final segments = narrationText
        .split('\n')
        .map((s) => s.trim())
        .where((s) => s.isNotEmpty)
        .toList();
    // Bölüm araları için konuşmaya eşdeğer duraklama payı (karakter).
    const pause = 12;
    final weights = [for (final s in segments) s.length + pause];
    final total = weights.fold<int>(0, (a, b) => a + b);
    final starts = <double>[];
    final cues = <IntroCue>[];
    var acc = 0;
    for (final (i, seg) in segments.indexed) {
      final segStart = kNarrationStart + script.narration * acc / total;
      final segEnd =
          kNarrationStart + script.narration * (acc + weights[i]) / total;
      starts.add(segStart);
      cues.addAll(_sentences(seg, segStart, segEnd));
      acc += weights[i];
    }
    return IntroTimeline._(script, starts, cues);
  }

  IntroTimeline._(this.script, this._segmentStarts, this.cues);

  final IntroScript script;
  final List<double> _segmentStarts;
  final List<IntroCue> cues;

  static const length = 30.0;

  double _segment(int i, double fallback) =>
      i < _segmentStarts.length ? _segmentStarts[i] : fallback;

  /// Felaket anı (çarpma, parlama, sarsıntı).
  double get impactAt => _segment(1, 8);

  /// Sığınak bölümü (ön plan girer).
  double get shelterAt => _segment(2, 15);

  /// Kapı bölümü (ortak kapı görseli).
  double get doorAt => _segment(3, 20);

  /// Anlatımın bittiği an.
  double get narrationEnd => kNarrationStart + script.narration;

  /// `t` anında gösterilecek altyazı (yoksa `null`).
  String? cueAt(double t) {
    for (final c in cues) {
      if (t >= c.start && t < c.end) return c.text;
    }
    return null;
  }

  // --- görsel eğriler (0–1) ---

  double titleOpacity(double t) =>
      _ramp(t, 0.3, 1.2) * (1 - _ramp(t, kNarrationStart, 2.8));

  double backOpacity(double t) => _ramp(t, 1.2, 2.6);

  /// Kamera: yavaş yaklaşma (1.22 → 1.04), kapıya kadar.
  double backScale(double t) => 1.22 - 0.18 * _ease(_ramp(t, 1.5, doorAt));

  double midOpacity(double t) => _ramp(t, 2, 3.2);

  /// Orta plan aşağıdan yerine kayar (ekran yüksekliği oranı).
  double midShift(double t) => 0.06 * (1 - _ease(_ramp(t, 2, shelterAt)));

  /// Ön plan: sığınak bölümünde aşağıdan girer (1 → 0).
  double frontShift(double t) =>
      0.35 * (1 - _ease(_ramp(t, shelterAt, shelterAt + 2.2)));

  double frontOpacity(double t) => _ramp(t, shelterAt, shelterAt + 1.2);

  /// Sığınağa yaklaşırken arka plan kararır.
  double dim(double t) => 0.35 * _ramp(t, shelterAt, shelterAt + 2);

  double doorOpacity(double t) => _ramp(t, doorAt, doorAt + 1.4);

  double doorScale(double t) =>
      1 + 0.14 * _ease(_ramp(t, doorAt, length - 2.5));

  /// Sonda siyaha geçiş.
  double fadeOut(double t) => _ramp(t, length - 3, length - 0.6);

  /// Çarpma parlaması.
  double flash(double t, {required bool reduceMotion}) {
    if (reduceMotion) return 0;
    final dt = t - impactAt;
    if (dt < 0 || dt > 0.9) return 0;
    return script.flashStrength * (1 - dt / 0.9);
  }

  /// Ekran sarsıntısı (piksel).
  Offset shake(double t, {required bool reduceMotion}) {
    if (reduceMotion) return Offset.zero;
    final dt = t - impactAt;
    if (dt < 0 || dt > 1.3) return Offset.zero;
    final amp = 14 * (1 - dt / 1.3);
    return Offset(amp * math.sin(dt * 71), amp * 0.7 * math.cos(dt * 53));
  }

  /// Parçacık yoğunluğu: çarpmadan önce hafif, sonra tam.
  double particleIntensity(double t) =>
      0.35 * _ramp(t, 2, 3) + 0.65 * _ramp(t, impactAt, impactAt + 1.5);

  HeroPose hero(double t) {
    final s = script;
    final i = impactAt;
    final c = s.heroCenter;
    switch (s.hero) {
      case HeroMotion.flash:
        final grow = _ease(_ramp(t, i - 0.2, i + 2));
        final fade = 1 - 0.6 * _ramp(t, shelterAt - 1, shelterAt + 1);
        return HeroPose(
          opacity: _ramp(t, i - 0.2, i + 0.2) * fade,
          center: c,
          scale: 0.25 + 0.75 * grow,
        );
      case HeroMotion.fall:
        final p = _ease(_ramp(t, i - 2.6, i));
        return HeroPose(
          opacity: _ramp(t, i - 2.6, i - 2.2) * (1 - _ramp(t, i, i + 0.35)),
          center: Offset.lerp(const Offset(1.15, -0.2), c, p)!,
          scale: 0.55 + 0.45 * p,
        );
      case HeroMotion.flicker:
        final flick = 0.8 + 0.2 * math.sin(t * 23) * math.sin(t * 7.3);
        return HeroPose(
          opacity:
              _ramp(t, i - 1.5, i) *
              (1 - _ramp(t, shelterAt - 0.5, shelterAt + 0.8)) *
              flick,
          center: c,
          scale: 1,
        );
      case HeroMotion.hover:
        final enter = _ease(_ramp(t, i - 1.5, i + 1));
        final leave = _ease(_ramp(t, shelterAt - 0.5, shelterAt + 1.5));
        final x = -0.4 + (c.dx + 0.4) * enter + 1.0 * leave;
        return HeroPose(
          opacity: _ramp(t, i - 1.5, i - 1),
          center: Offset(x, c.dy + 0.012 * math.sin(t * 3.1)),
          scale: 0.9 + 0.1 * enter,
        );
      case HeroMotion.rise:
        final up = _ease(_ramp(t, i - 3, i));
        final pulse = 1 + 0.03 * math.sin(t * 4);
        return HeroPose(
          opacity:
              _ramp(t, i - 3, i - 2) *
              (1 - 0.5 * _ramp(t, shelterAt - 1, shelterAt + 1)),
          center: Offset(c.dx, c.dy + 0.45 * (1 - up)),
          scale: (0.8 + 0.2 * up) * pulse,
        );
    }
  }

  static List<IntroCue> _sentences(String seg, double start, double end) {
    final parts = RegExp(r'[^.?!…]+(?:[.?!…]+|$)')
        .allMatches(seg)
        .map((m) => m.group(0)!.trim())
        .where((s) => s.isNotEmpty)
        .toList();
    if (parts.isEmpty) return const [];
    final total = parts.fold<int>(0, (a, s) => a + s.length);
    final out = <IntroCue>[];
    var acc = 0;
    for (final p in parts) {
      final a = start + (end - start) * acc / total;
      acc += p.length;
      final b = start + (end - start) * acc / total;
      out.add(IntroCue(a, b, p));
    }
    return out;
  }
}

/// Animasyonun başlayacağı an: aşamanın geçen süresi. Oyuna sonradan
/// katılan ya da yeniden bağlanan oyuncu ortadan başlar.
double introStartAt(Duration? remaining) {
  if (remaining == null) return 0;
  final elapsed =
      (kIntroLength - remaining).inMilliseconds /
      Duration.millisecondsPerSecond;
  return elapsed.clamp(0, IntroTimeline.length).toDouble();
}

/// Bu kadar az süre kaldıysa animasyon hiç gösterilmez.
const kIntroMinRemaining = Duration(seconds: 2);

double _ramp(double t, double a, double b) {
  if (b <= a) return t >= b ? 1 : 0;
  return ((t - a) / (b - a)).clamp(0, 1).toDouble();
}

double _ease(double x) => x * x * (3 - 2 * x);
