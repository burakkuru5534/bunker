import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:just_audio/just_audio.dart';

import 'package:siginak/core/l10n/l10n.dart';

/// Oyun sesleri (lobi müziği, senaryo tanıtımı) kapalı mı? Uygulama açık
/// kaldıkça hatırlanır.
final soundMutedProvider = NotifierProvider<SoundMuted, bool>(SoundMuted.new);

class SoundMuted extends Notifier<bool> {
  @override
  bool build() => false;

  void toggle() => state = !state;
}

/// Ses aç / kapat düğmesi (lobi çubuğu, senaryo tanıtımı).
class SoundToggleButton extends ConsumerWidget {
  const SoundToggleButton({this.color, super.key});

  final Color? color;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final muted = ref.watch(soundMutedProvider);
    final l = context.l10n;
    return IconButton(
      key: const ValueKey('sound-toggle'),
      tooltip: muted ? l.gameIntroUnmute : l.gameIntroMute,
      color: color,
      onPressed: () => ref.read(soundMutedProvider.notifier).toggle(),
      icon: Icon(muted ? Icons.volume_off_rounded : Icons.volume_up_rounded),
    );
  }
}

/// Döngüde çalan arka plan müziği.
abstract class LoopingMusic {
  Future<void> play();
  Future<void> pause();
  void setMuted({required bool muted});
  Future<void> dispose();
}

/// Lobi müziği örneği üretir (testlerde sahtesiyle değişir).
final lobbyMusicFactoryProvider = Provider<LoopingMusic Function()>(
  (ref) =>
      () => JustAudioLoopingMusic('assets/scenarios/music.ogg', volume: 0.3),
);

/// `just_audio` ile döngü. Dosya açılamazsa sessiz devam eder.
class JustAudioLoopingMusic implements LoopingMusic {
  JustAudioLoopingMusic(this.asset, {required this.volume});

  final String asset;
  final double volume;
  final _player = AudioPlayer();
  Future<void>? _loading;
  bool _muted = false;
  bool _disposed = false;

  Future<void> _load() => _loading ??= () async {
    try {
      await _player.setAsset(asset);
      await _player.setLoopMode(LoopMode.one);
      await _player.setVolume(_muted ? 0 : volume);
    } on Object catch (e) {
      debugPrint('Müzik açılamadı ($asset): $e');
    }
  }();

  @override
  Future<void> play() async {
    await _load();
    if (!_disposed) unawaited(_player.play());
  }

  @override
  Future<void> pause() async {
    if (!_disposed) await _player.pause();
  }

  @override
  void setMuted({required bool muted}) {
    _muted = muted;
    if (!_disposed) unawaited(_player.setVolume(muted ? 0 : volume));
  }

  @override
  Future<void> dispose() async {
    _disposed = true;
    await _player.dispose();
  }
}

/// Altındaki ekran görünürken döngü müziği çalar: uygulama arka plana
/// geçince duraklar, öne gelince sürer; ses kapalıysa sessizdir.
class LoopingMusicScope extends ConsumerStatefulWidget {
  const LoopingMusicScope({
    required this.factory,
    required this.child,
    super.key,
  });

  final Provider<LoopingMusic Function()> factory;
  final Widget child;

  @override
  ConsumerState<LoopingMusicScope> createState() => _LoopingMusicScopeState();
}

class _LoopingMusicScopeState extends ConsumerState<LoopingMusicScope> {
  late final LoopingMusic _music;
  late final AppLifecycleListener _lifecycle;

  @override
  void initState() {
    super.initState();
    _music = ref.read(widget.factory)()
      ..setMuted(muted: ref.read(soundMutedProvider));
    unawaited(_music.play());
    _lifecycle = AppLifecycleListener(
      onHide: () => unawaited(_music.pause()),
      onShow: () => unawaited(_music.play()),
    );
  }

  @override
  void dispose() {
    _lifecycle.dispose();
    unawaited(_music.dispose());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    ref.listen<bool>(
      soundMutedProvider,
      (_, muted) => _music.setMuted(muted: muted),
    );
    return widget.child;
  }
}
