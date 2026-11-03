import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/session_store.dart';

/// Güvenli depo; testlerde `InMemorySecureStore` ile override edilir.
final secureStoreProvider = Provider<SecureStore>(
  (ref) => FlutterSecureStore(),
);

/// Uygulama boyu tek oturum deposu.
final sessionStoreProvider = Provider<SessionStore>(
  (ref) => SessionStore(ref.watch(secureStoreProvider)),
);
