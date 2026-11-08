import 'package:siginak/features/game/presentation/intro/intro_audio.dart';
import 'package:siginak/features/game/presentation/intro/intro_timeline.dart';

/// Ses çalmayan tanıtım sesi; çağrıları kaydeder.
class FakeIntroAudio implements IntroAudio {
  FakeIntroAudio([this.log]);

  final List<FakeIntroAudio>? log;
  String? scenarioId;
  double? startedAt;
  final muted = <bool>[];
  bool disposed = false;

  @override
  Future<void> start(IntroTimeline timeline, double at) async {
    log?.add(this);
    scenarioId = timeline.script.id;
    startedAt = at;
  }

  @override
  void setMuted({required bool muted}) => this.muted.add(muted);

  @override
  Future<void> dispose() async => disposed = true;
}
