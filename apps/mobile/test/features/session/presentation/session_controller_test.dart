import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/network/auth_token_provider.dart';
import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/session_store.dart';
import 'package:siginak/core/storage/storage_providers.dart';
import 'package:siginak/features/session/session.dart';

/// Okunamayan depo (ör. Keystore anahtarı kaybolmuş).
class BrokenStore implements SecureStore {
  @override
  Future<String?> read(String key) => throw Exception('keystore');
  @override
  Future<void> write(String key, String value) async {}
  @override
  Future<void> delete(String key) async {}
}

void main() {
  final valid = StoredSession(
    token: 'tok',
    expiresAt: DateTime.now().add(const Duration(days: 30)),
    userId: 'u1',
    nickname: 'Ayşe',
  );

  ProviderContainer container(SecureStore store) {
    final c = ProviderContainer(
      overrides: [secureStoreProvider.overrideWithValue(store)],
    );
    addTearDown(c.dispose);
    return c;
  }

  Future<SessionState> settle(ProviderContainer c) async {
    c.read(sessionControllerProvider);
    await Future<void>.delayed(Duration.zero);
    return c.read(sessionControllerProvider);
  }

  test('açılışta unknown, depo boşsa unauthenticated', () async {
    final c = container(InMemorySecureStore());

    expect(c.read(sessionControllerProvider).status, SessionStatus.unknown);
    expect((await settle(c)).status, SessionStatus.unauthenticated);
  });

  test('saklı geçerli oturum varsa authenticated', () async {
    final store = InMemorySecureStore();
    await SessionStore(store).save(valid);
    final c = container(store);

    final state = await settle(c);

    expect(state.status, SessionStatus.authenticated);
    expect(state.userId, 'u1');
    expect(state.nickname, 'Ayşe');
  });

  test('depo okunamazsa unauthenticated', () async {
    final c = container(BrokenStore());

    expect((await settle(c)).status, SessionStatus.unauthenticated);
  });

  test('signIn saklar ve token okuyucuya yansır; signOut siler', () async {
    final store = InMemorySecureStore();
    final c = container(store);
    await settle(c);
    final controller = c.read(sessionControllerProvider.notifier);

    await controller.signIn(valid);
    expect(
      c.read(sessionControllerProvider).status,
      SessionStatus.authenticated,
    );
    expect(store.values, contains(SessionStore.storageKey));
    expect(await c.read(authTokenReaderProvider)(), 'tok');

    await controller.signOut();
    expect(
      c.read(sessionControllerProvider).status,
      SessionStatus.unauthenticated,
    );
    expect(store.values, isEmpty);
    expect(await c.read(authTokenReaderProvider)(), isNull);
  });
}
