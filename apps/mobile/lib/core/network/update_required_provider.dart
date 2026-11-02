import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Sunucu uygulama sürümünü desteklemediğini bildirdiğinde (`426` /
/// WS `4026`) `true` olur ve router kullanıcıyı `/update` ekranına kilitler.
///
/// Ağ katmanı (MOB-006 / MOB-007) bu durumu ayarlayacaktır.
class UpdateRequiredNotifier extends Notifier<bool> {
  @override
  bool build() => false;

  void markRequired() => state = true;
}

final updateRequiredProvider = NotifierProvider<UpdateRequiredNotifier, bool>(
  UpdateRequiredNotifier.new,
);
