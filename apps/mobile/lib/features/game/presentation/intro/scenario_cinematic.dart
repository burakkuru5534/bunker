import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/l10n/l10n.dart';
import 'package:siginak/core/theme/app_spacing.dart';
import 'package:siginak/core/theme/app_typography.dart';
import 'package:siginak/features/game/presentation/intro/intro_audio.dart';
import 'package:siginak/features/game/presentation/intro/intro_particles.dart';
import 'package:siginak/features/game/presentation/intro/intro_timeline.dart';

/// Senaryonun anlatım metni (altyazı); bilinmeyen senaryoda `null`.
String? introNarration(AppLocalizations l, String scenarioId) =>
    switch (scenarioId) {
      'nuclear_winter' => l.introNarrationNuclearWinter,
      'red_cough' => l.introNarrationRedCough,
      'meteor' => l.introNarrationMeteor,
      'silent_machines' => l.introNarrationSilentMachines,
      'solar_storm' => l.introNarrationSolarStorm,
      _ => null,
    };

/// Senaryo tanıtımı: katmanlı görseller, kamera hareketi, parçacıklar,
/// çarpma anı, sığınak ve ortak kapı; anlatım, müzik ve altyazı
/// (docs/ui/scenario-intro-assets.md).
///
/// Tamamen [startAt] ve geçen zamanın fonksiyonudur; sonradan katılan oyuncu
/// ortadan başlar. Oyun sunucunun süresiyle ilerler: "Geç" yalnızca bu
/// oyuncunun ekranındaki animasyonu kapatır.
class ScenarioCinematic extends ConsumerStatefulWidget {
  const ScenarioCinematic({
    required this.timeline,
    required this.title,
    required this.startAt,
    required this.onSkip,
    super.key,
  });

  final IntroTimeline timeline;
  final String title;

  /// Başlangıç anı (sn).
  final double startAt;
  final VoidCallback onSkip;

  @override
  ConsumerState<ScenarioCinematic> createState() => _ScenarioCinematicState();
}

class _ScenarioCinematicState extends ConsumerState<ScenarioCinematic>
    with SingleTickerProviderStateMixin {
  late final AnimationController _clock;
  late final IntroAudio _audio;
  bool _precached = false;

  IntroTimeline get _tl => widget.timeline;

  @override
  void initState() {
    super.initState();
    _clock = AnimationController(
      vsync: this,
      duration: kIntroLength,
      value: widget.startAt / IntroTimeline.length,
    );
    unawaited(_clock.forward());
    _audio = ref.read(introAudioFactoryProvider)()
      ..setMuted(muted: ref.read(introMutedProvider));
    unawaited(_audio.start(_tl, widget.startAt));
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_precached) return;
    _precached = true;
    final s = _tl.script;
    for (final a in [
      s.asset('back.webp'),
      s.asset('mid.webp'),
      s.asset('hero.webp'),
      s.asset('front.webp'),
      'assets/scenarios/door.webp',
    ]) {
      unawaited(precacheImage(AssetImage(a), context, onError: (_, _) {}));
    }
  }

  @override
  void dispose() {
    _clock.dispose();
    unawaited(_audio.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    ref.listen<bool>(
      introMutedProvider,
      (_, muted) => _audio.setMuted(muted: muted),
    );
    final muted = ref.watch(introMutedProvider);
    final l = context.l10n;
    final reduceMotion = MediaQuery.disableAnimationsOf(context);

    return ColoredBox(
      key: const ValueKey('scenario-cinematic'),
      color: Colors.black,
      child: Stack(
        fit: StackFit.expand,
        children: [
          AnimatedBuilder(
            animation: _clock,
            builder: (context, _) => _Frame(
              timeline: _tl,
              title: widget.title,
              t: _clock.value * IntroTimeline.length,
              reduceMotion: reduceMotion,
            ),
          ),
          SafeArea(
            child: Align(
              alignment: Alignment.topRight,
              child: Padding(
                padding: const EdgeInsets.all(AppSpacing.s2),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    IconButton(
                      key: const ValueKey('intro-mute'),
                      tooltip: muted ? l.gameIntroUnmute : l.gameIntroMute,
                      color: Colors.white,
                      onPressed: () =>
                          ref.read(introMutedProvider.notifier).toggle(),
                      icon: Icon(
                        muted
                            ? Icons.volume_off_rounded
                            : Icons.volume_up_rounded,
                      ),
                    ),
                    TextButton(
                      key: const ValueKey('intro-skip'),
                      style: TextButton.styleFrom(
                        foregroundColor: Colors.white,
                        minimumSize: const Size(48, 48),
                      ),
                      onPressed: widget.onSkip,
                      child: Text(l.gameIntroSkip),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _Frame extends StatelessWidget {
  const _Frame({
    required this.timeline,
    required this.title,
    required this.t,
    required this.reduceMotion,
  });

  final IntroTimeline timeline;
  final String title;
  final double t;
  final bool reduceMotion;

  @override
  Widget build(BuildContext context) {
    final tl = timeline;
    final s = tl.script;
    final theme = Theme.of(context).textTheme;
    final cue = tl.cueAt(t);
    final hero = tl.hero(t);
    final flash = tl.flash(t, reduceMotion: reduceMotion);

    return LayoutBuilder(
      builder: (context, box) {
        final size = box.biggest;
        final heroW = size.width * s.heroWidth * hero.scale;
        return Stack(
          fit: StackFit.expand,
          children: [
            Transform.translate(
              offset: tl.shake(t, reduceMotion: reduceMotion),
              child: Stack(
                fit: StackFit.expand,
                children: [
                  _layer(
                    s.asset('back.webp'),
                    opacity: tl.backOpacity(t),
                    scale: tl.backScale(t),
                  ),
                  _layer(
                    s.asset('mid.webp'),
                    opacity: tl.midOpacity(t),
                    shift: tl.midShift(t),
                    alignment: Alignment.bottomCenter,
                  ),
                  if (hero.opacity > 0)
                    Positioned(
                      left: hero.center.dx * size.width - heroW / 2,
                      top: hero.center.dy * size.height - heroW / 2,
                      width: heroW,
                      height: heroW,
                      child: Opacity(
                        opacity: hero.opacity.clamp(0, 1),
                        child: _image(s.asset('hero.webp'), BoxFit.contain),
                      ),
                    ),
                  IgnorePointer(
                    child: CustomPaint(
                      painter: IntroParticlesPainter(
                        kind: s.particles,
                        t: t,
                        intensity: tl.particleIntensity(t),
                      ),
                    ),
                  ),
                  ColoredBox(color: Colors.black.withValues(alpha: tl.dim(t))),
                  _layer(
                    s.asset('front.webp'),
                    opacity: tl.frontOpacity(t),
                    shift: tl.frontShift(t),
                    alignment: Alignment.bottomCenter,
                  ),
                  _layer(
                    'assets/scenarios/door.webp',
                    opacity: tl.doorOpacity(t),
                    scale: tl.doorScale(t),
                  ),
                ],
              ),
            ),
            if (flash > 0) ColoredBox(color: s.flash.withValues(alpha: flash)),
            // Altyazı okunabilsin diye alt kısım karartılır.
            const IgnorePointer(
              child: DecoratedBox(
                decoration: BoxDecoration(
                  gradient: LinearGradient(
                    begin: Alignment.topCenter,
                    end: Alignment.bottomCenter,
                    stops: [0.6, 1],
                    colors: [Colors.transparent, Color(0xCC000000)],
                  ),
                ),
              ),
            ),
            Center(
              child: Opacity(
                opacity: tl.titleOpacity(t),
                child: Padding(
                  padding: const EdgeInsets.all(AppSpacing.screen),
                  child: Text(
                    turkishUpperCase(title),
                    textAlign: TextAlign.center,
                    style: theme.displaySmall?.copyWith(color: Colors.white),
                  ),
                ),
              ),
            ),
            if (cue != null)
              Align(
                alignment: Alignment.bottomCenter,
                child: SafeArea(
                  minimum: const EdgeInsets.fromLTRB(
                    AppSpacing.screen,
                    0,
                    AppSpacing.screen,
                    AppSpacing.s6,
                  ),
                  child: Text(
                    cue,
                    key: const ValueKey('intro-subtitle'),
                    textAlign: TextAlign.center,
                    maxLines: 4,
                    overflow: TextOverflow.ellipsis,
                    style: theme.titleMedium?.copyWith(
                      color: Colors.white,
                      shadows: const [
                        Shadow(blurRadius: 8),
                        Shadow(blurRadius: 2),
                      ],
                    ),
                  ),
                ),
              ),
            IgnorePointer(
              child: ColoredBox(
                color: Colors.black.withValues(alpha: tl.fadeOut(t)),
              ),
            ),
          ],
        );
      },
    );
  }

  static Widget _layer(
    String asset, {
    required double opacity,
    double scale = 1,
    double shift = 0,
    Alignment alignment = Alignment.center,
  }) {
    if (opacity <= 0) return const SizedBox.shrink();
    return Opacity(
      opacity: opacity.clamp(0, 1),
      child: FractionalTranslation(
        translation: Offset(0, shift),
        child: Transform.scale(
          scale: scale,
          alignment: alignment,
          child: _image(asset, BoxFit.cover, alignment: alignment),
        ),
      ),
    );
  }

  static Widget _image(
    String asset,
    BoxFit fit, {
    Alignment alignment = Alignment.center,
  }) => Image.asset(
    asset,
    fit: fit,
    alignment: alignment,
    gaplessPlayback: true,
    excludeFromSemantics: true,
    errorBuilder: (_, _, _) => const SizedBox.shrink(),
  );
}
