import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';

import 'package:siginak/core/storage/secure_store.dart';
import 'package:siginak/core/storage/session_store.dart';

/// Okuma sayısını sayan depo.
class CountingStore extends InMemorySecureStore {
  int reads = 0;

  @override
  Future<String?> read(String key) {
    reads++;
    return super.read(key);
  }
}

void main() {
  final now = DateTime.utc(2026, 11, 3, 12);
  StoredSession session({DateTime? expiresAt}) => StoredSession(
    token: 'secret-token',
    expiresAt: expiresAt ?? now.add(const Duration(days: 90)),
    userId: 'u1',
    nickname: 'Ayşe',
  );

  test('kaydedilen oturum geri okunur (yeni örnekle de)', () async {
    final store = InMemorySecureStore();
    await SessionStore(store, clock: () => now).save(session());

    final fresh = SessionStore(store, clock: () => now);
    expect(await fresh.read(), session());
    expect(await fresh.readToken(), 'secret-token');
  });

  test('depo yalnızca ilk okumada okunur (önbellek)', () async {
    final store = CountingStore();
    await SessionStore(store, clock: () => now).save(session());
    final s = SessionStore(store, clock: () => now);

    await s.readToken();
    await s.readToken();
    await s.readToken();

    expect(store.reads, 1);
  });

  test('süresi dolmuş oturum null döner ve silinir', () async {
    final store = InMemorySecureStore();
    await SessionStore(
      store,
      clock: () => now,
    ).save(session(expiresAt: now.subtract(const Duration(seconds: 1))));

    final s = SessionStore(store, clock: () => now);
    expect(await s.read(), isNull);
    expect(store.values, isEmpty);
  });

  test('bozuk veya eksik veri silinir', () async {
    for (final raw in [
      '{bozuk json',
      jsonEncode({'token': 'x'}),
      jsonEncode({
        'token': '',
        'expires_at': '2030-01-01T00:00:00Z',
        'user_id': 'u',
        'nickname': 'n',
      }),
      jsonEncode({
        'token': 't',
        'expires_at': 'tarih değil',
        'user_id': 'u',
        'nickname': 'n',
      }),
      jsonEncode([1, 2]),
    ]) {
      final store = InMemorySecureStore({SessionStore.storageKey: raw});
      expect(
        await SessionStore(store, clock: () => now).read(),
        isNull,
        reason: raw,
      );
      expect(store.values, isEmpty, reason: raw);
    }
  });

  test('clear oturumu siler', () async {
    final store = InMemorySecureStore();
    final s = SessionStore(store, clock: () => now);
    await s.save(session());

    await s.clear();

    expect(await s.read(), isNull);
    expect(store.values, isEmpty);
  });

  test('toString token içermez', () {
    expect(session().toString(), isNot(contains('secret-token')));
  });
}
