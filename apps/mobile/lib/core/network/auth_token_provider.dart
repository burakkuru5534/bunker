import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/network/api_client.dart';

/// Geçerli oturum token'ını okuyan fonksiyon.
///
/// Şimdilik her zaman `null` döner; güvenli depolama (MOB-010) bu
/// provider'ı gerçek okuyucuyla değiştirecektir.
final authTokenReaderProvider = Provider<TokenReader>(
  (ref) =>
      () async => null,
);

/// Sunucu 401 döndürdüğünde çağrılır. Oturum katmanı (AUTH-002) bu
/// provider'ı override ederek oturumu temizleyecektir.
final unauthorizedHandlerProvider = Provider<void Function()>((ref) => () {});
