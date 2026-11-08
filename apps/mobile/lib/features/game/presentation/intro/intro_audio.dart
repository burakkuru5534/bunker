import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:just_audio/just_audio.dart';

import 'package:siginak/features/game/presentation/intro/intro_timeline.dart';

/// Tanıtımın sesleri: müzik (baştan), anlatım ([kNarrationStart]'ta) ve
/// çarpma efekti. Her tanıtım kendi örneğini açar ve kapatır.
abstract class IntroAudio {
  /// [at] (sn) anından başlatır: geçmişte kalan sesler atlanır, süren sesler
  /// ilgili saniyeden çalar.
  Future<void> start(IntroTimeline timeline, double at);

  /// Sessize alır / açar (zamanlama sürer, yalnızca ses düzeyi değişir).
  void setMuted({required bool muted});

  Future<void> dispose();
}

/// Ses kapalı mı? Uygulama açık kaldıkça hatırlanır.
final introMutedProvider = NotifierProvider<IntroMuted, bool>(IntroMuted.new);

class IntroMuted extends Notifier<bool> {
  @override
  bool build() => false;

  void toggle() => state = !state;
}

/// Her tanıtım için yeni ses örneği üretir (testlerde sahtesiyle değişir).
final introAudioFactoryProvider = Provider<IntroAudio Function()>(
  (ref) => JustAudioIntroAudio.new,
);

/// `just_audio` ile çalan uygulama. Ses dosyası açılamazsa sessiz devam eder
/// (animasyon sesten bağımsızdır).
class JustAudioIntroAudio implements IntroAudio {
  static const musicAsset = 'assets/scenarios/music.ogg';
  static const musicVolume = 0.45;

  final _music = AudioPlayer();
  final _narration = AudioPlayer();
  final _impact = AudioPlayer();
  final _timers = <Timer>[];
  bool _muted = false;
  bool _disposed = false;

  @override
  Future<void> start(IntroTimeline timeline, double at) async {
    final s = timeline.script;
    await _play(_music, musicAsset, at, at, musicVolume);
    _schedule(_narration, s.asset('narration.ogg'), kNarrationStart, at, 1);
    _schedule(_impact, s.asset('impact.ogg'), timeline.impactAt, at, 0.9);
  }

  void _schedule(
    AudioPlayer p,
    String asset,
    double startsAt,
    double now,
    double volume,
  ) {
    if (now <= startsAt) {
      final wait = Duration(milliseconds: ((startsAt - now) * 1000).round());
      _timers.add(Timer(wait, () => unawaited(_play(p, asset, 0, 0, volume))));
    } else {
      unawaited(_play(p, asset, now - startsAt, now, volume));
    }
  }

  Future<void> _play(
    AudioPlayer p,
    String asset,
    double offset,
    double now,
    double volume,
  ) async {
    if (_disposed) return;
    try {
      final length = await p.setAsset(asset);
      if (_disposed) return;
      if (length != null && offset * 1000 >= length.inMilliseconds) return;
      await p.setVolume(_muted ? 0 : volume);
      await p.seek(Duration(milliseconds: (offset * 1000).round()));
      unawaited(p.play());
    } on Object catch (e) {
      debugPrint('Tanıtım sesi çalınamadı ($asset): $e');
    }
  }

  @override
  void setMuted({required bool muted}) {
    _muted = muted;
    unawaited(_music.setVolume(muted ? 0 : musicVolume));
    unawaited(_narration.setVolume(muted ? 0 : 1));
    unawaited(_impact.setVolume(muted ? 0 : 0.9));
  }

  @override
  Future<void> dispose() async {
    _disposed = true;
    for (final t in _timers) {
      t.cancel();
    }
    await Future.wait([
      _music.dispose(),
      _narration.dispose(),
      _impact.dispose(),
    ]);
  }
}
