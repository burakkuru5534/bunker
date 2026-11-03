import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/network/api_client.dart';
import 'package:siginak/core/storage/storage_providers.dart';

/// Geçerli oturum token'ını güvenli depodan okuyan fonksiyon.
final authTokenReaderProvider = Provider<TokenReader>((ref) {
  final store = ref.watch(sessionStoreProvider);
  return store.readToken;
});

/// Sunucu 401 / WS 4001 döndürdüğünde çağrılır.
///
/// Varsayılanı hiçbir şey yapmaz; uygulama katmanı (`main.dart`) bunu
/// oturumu kapatacak şekilde override eder (core, feature'lara bağımlı olamaz).
final unauthorizedHandlerProvider = Provider<void Function()>((ref) => () {});
