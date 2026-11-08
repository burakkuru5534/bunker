import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/config/app_config_provider.dart';
import 'package:siginak/core/network/api_client_provider.dart';
import 'package:siginak/core/network/api_exception.dart';
import 'package:siginak/core/network/auth_token_provider.dart';
import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/session_store.dart';
import 'package:siginak/core/storage/storage_providers.dart';
import 'package:siginak/features/session/session.dart';

import '../../../helpers/fake_backend.dart';
import '../../../helpers/fake_http_adapter.dart';
import '../../../helpers/pump_app.dart';

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
  StoredSession stored({Duration left = const Duration(days: 30)}) =>
      StoredSession(
        token: 'tok',
        expiresAt: DateTime.now().add(left),
        userId: 'u1',
        nickname: 'Ayşe',
      );

  late FakeBackend backend;
  setUp(() => backend = FakeBackend());

  ProviderContainer container(SecureStore store) {
    final c = ProviderContainer(
      overrides: [
        appConfigProvider.overrideWithValue(testConfig()),
        secureStoreProvider.overrideWithValue(store),
        apiHttpAdapterProvider.overrideWithValue(backend.adapter),
      ],
    );
    addTearDown(c.dispose);
    return c;
  }

  /// Açılış doğrulamasının bitmesini bekler.
  Future<SessionState> settle(ProviderContainer c) async {
    c.read(sessionControllerProvider);
    for (var i = 0; i < 20; i++) {
      await Future<void>.delayed(Duration.zero);
    }
    return c.read(sessionControllerProvider);
  }

  Future<InMemorySecureStore> storeWith(StoredSession s) async {
    final store = InMemorySecureStore();
    await SessionStore(store).save(s);
    return store;
  }

  test(
    'açılışta unknown, depo boşsa sunucuya gitmeden unauthenticated',
    () async {
      final c = container(InMemorySecureStore());

      expect(c.read(sessionControllerProvider).status, SessionStatus.unknown);
      expect((await settle(c)).status, SessionStatus.unauthenticated);
      expect(backend.calls, isEmpty);
    },
  );

  test('saklı oturum GET /v1/me ile doğrulanır; aktif oda gelir', () async {
    backend.activeRoomId = 'room-1';
    final c = container(await storeWith(stored()));

    final state = await settle(c);

    expect(state.status, SessionStatus.authenticated);
    expect(state.userId, 'u1');
    expect(state.nickname, 'Ayşe');
    expect(state.activeRoomId, 'room-1');
    expect(backend.calls, ['GET /v1/me']);
    expect(
      backend.adapter.requests.single.headers['Authorization'],
      'Bearer tok',
    );
  });

  test('sunucudaki takma ad farklıysa depo güncellenir', () async {
    backend.nickname = 'Yeni Ad';
    final store = await storeWith(stored());
    final c = container(store);

    expect((await settle(c)).nickname, 'Yeni Ad');
    expect((await SessionStore(store).read())!.nickname, 'Yeni Ad');
  });

  test('401 → oturum silinir, unauthenticated', () async {
    backend.on(
      'GET',
      '/v1/me',
      (r) async => jsonResponse(apiError('UNAUTHORIZED'), 401),
    );
    final store = await storeWith(stored());
    final c = container(store);

    expect((await settle(c)).status, SessionStatus.unauthenticated);
    expect(store.values, isEmpty);
  });

  test(
    'ağ hatası → restoreFailed; oturum korunur, tekrar denenebilir',
    () async {
      backend.on('GET', '/v1/me', networkFailure);
      final store = await storeWith(stored());
      final c = container(store);

      final failed = await settle(c);
      expect(failed.status, SessionStatus.unknown);
      expect(failed.restoreError, isA<ApiException>());
      expect(store.values, isNotEmpty);

      backend.on(
        'GET',
        '/v1/me',
        (r) async => jsonResponse({
          'id': 'u1',
          'nickname': 'Ayşe',
          'active_room_id': null,
        }, 200),
      );
      await c.read(sessionControllerProvider.notifier).retryRestore();

      final ok = c.read(sessionControllerProvider);
      expect(ok.status, SessionStatus.authenticated);
      expect(ok.restoreError, isNull);
    },
  );

  test('bozuk /me yanıtı → restoreFailed', () async {
    backend.on('GET', '/v1/me', (r) async => jsonResponse({'id': 1}, 200));
    final c = container(await storeWith(stored()));

    expect((await settle(c)).restoreError, isA<ApiException>());
  });

  test('bitişine 14 günden az kalan token yenilenir', () async {
    final newExpiry = DateTime.utc(2030);
    backend.on(
      'POST',
      '/v1/sessions/refresh',
      (r) async => jsonResponse({
        'token': 'tok-new',
        'expires_at': newExpiry.toIso8601String(),
      }, 200),
    );
    final store = await storeWith(stored(left: const Duration(days: 3)));
    final c = container(store);

    expect((await settle(c)).status, SessionStatus.authenticated);
    expect(backend.calls, ['GET /v1/me', 'POST /v1/sessions/refresh']);
    final saved = (await SessionStore(store).read())!;
    expect(saved.token, 'tok-new');
    expect(saved.expiresAt, newExpiry);
  });

  test(
    'uzun süreli token yenilenmez; yenileme hatası oturumu bozmaz',
    () async {
      final c1 = container(await storeWith(stored()));
      await settle(c1);
      expect(backend.calls, ['GET /v1/me']);

      backend = FakeBackend()
        ..on('POST', '/v1/sessions/refresh', networkFailure);
      final store = await storeWith(stored(left: const Duration(days: 1)));
      final c2 = container(store);
      expect((await settle(c2)).status, SessionStatus.authenticated);
      expect((await SessionStore(store).read())!.token, 'tok');
    },
  );

  test('depo okunamazsa unauthenticated', () async {
    final c = container(BrokenStore());

    expect((await settle(c)).status, SessionStatus.unauthenticated);
  });

  test('signInAsGuest misafir hesap açar, saklar; signOut siler', () async {
    final store = InMemorySecureStore();
    final c = container(store);
    await settle(c);
    final controller = c.read(sessionControllerProvider.notifier);

    await controller.signInAsGuest('Burak');

    final state = c.read(sessionControllerProvider);
    expect(state.status, SessionStatus.authenticated);
    expect(state.nickname, 'Burak');
    expect(state.activeRoomId, isNull);
    expect(backend.adapter.requests.single.data, {'nickname': 'Burak'});
    expect(await c.read(authTokenReaderProvider)(), 'tok-guest');

    await controller.signOut();
    expect(
      c.read(sessionControllerProvider).status,
      SessionStatus.unauthenticated,
    );
    expect(store.values, isEmpty);
    expect(await c.read(authTokenReaderProvider)(), isNull);
  });

  test('signInAsGuest hatası fırlatılır, oturum açılmaz', () async {
    backend.on(
      'POST',
      '/v1/sessions/guest',
      (r) async => jsonResponse(
        apiError(
          'NICKNAME_INVALID',
          details: {
            'fields': {'nickname': 'profanity'},
          },
        ),
        400,
      ),
    );
    final store = InMemorySecureStore();
    final c = container(store);
    await settle(c);

    await expectLater(
      c.read(sessionControllerProvider.notifier).signInAsGuest('kötü'),
      throwsA(
        isA<ApiException>().having(
          (e) => e.fieldErrors['nickname'],
          'alan',
          'profanity',
        ),
      ),
    );
    expect(
      c.read(sessionControllerProvider).status,
      SessionStatus.unauthenticated,
    );
    expect(store.values, isEmpty);
  });

  test('changeNickname sunucuya yazar, depoyu ve durumu günceller', () async {
    final store = await storeWith(stored());
    final c = container(store);
    await settle(c);

    await c.read(sessionControllerProvider.notifier).changeNickname('Yeni');

    expect(backend.adapter.requests.last.data, {'nickname': 'Yeni'});
    expect(c.read(sessionControllerProvider).nickname, 'Yeni');
    expect((await SessionStore(store).read())!.nickname, 'Yeni');
  });

  test(
    'needs_nickname açılışta gelir; confirmNickname onaylar ve korur',
    () async {
      backend.needsNickname = true;
      final store = await storeWith(stored());
      final c = container(store);
      final restored = await settle(c);
      expect(restored.needsNickname, isTrue);

      final controller = c.read(sessionControllerProvider.notifier)
        ..setActiveRoom('r1');
      expect(c.read(sessionControllerProvider).needsNickname, isTrue);

      await controller.confirmNickname('Ayşe');
      final s = c.read(sessionControllerProvider);
      expect(s.needsNickname, isFalse);
      expect(s.activeRoomId, 'r1');
      expect(backend.calls.last, 'PATCH /v1/me');
    },
  );

  test('confirmNickname hatasında onay ekranı kalır', () async {
    backend
      ..needsNickname = true
      ..on(
        'PATCH',
        '/v1/me',
        (r) async => jsonResponse(apiError('NICKNAME_TAKEN'), 409),
      );
    final c = container(await storeWith(stored()));
    await settle(c);

    await expectLater(
      c.read(sessionControllerProvider.notifier).confirmNickname('Ayşe'),
      throwsA(isA<ApiException>()),
    );
    expect(c.read(sessionControllerProvider).needsNickname, isTrue);
  });

  test('changeNickname hatası fırlatılır, ad değişmez', () async {
    backend.on(
      'PATCH',
      '/v1/me',
      (r) async => jsonResponse(apiError('INVALID_PHASE'), 409),
    );
    final store = await storeWith(stored());
    final c = container(store);
    await settle(c);

    await expectLater(
      c.read(sessionControllerProvider.notifier).changeNickname('Yeni'),
      throwsA(isA<ApiException>()),
    );
    expect(c.read(sessionControllerProvider).nickname, 'Ayşe');
    expect((await SessionStore(store).read())!.nickname, 'Ayşe');
  });

  test(
    'setActiveRoom yalnızca oturum açıkken aktif odayı değiştirir',
    () async {
      final c = container(InMemorySecureStore());
      await settle(c);
      final controller = c.read(sessionControllerProvider.notifier)
        ..setActiveRoom('r1');
      expect(c.read(sessionControllerProvider).activeRoomId, isNull);

      await controller.signInAsGuest('Burak');
      controller.setActiveRoom('r1');
      expect(c.read(sessionControllerProvider).activeRoomId, 'r1');
      expect(c.read(sessionControllerProvider).nickname, 'Burak');
      controller.setActiveRoom(null);
      expect(c.read(sessionControllerProvider).activeRoomId, isNull);
    },
  );
}
