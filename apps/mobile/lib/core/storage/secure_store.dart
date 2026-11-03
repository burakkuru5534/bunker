import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Gizli değerler için anahtar–değer deposu soyutlaması.
abstract interface class SecureStore {
  Future<String?> read(String key);
  Future<void> write(String key, String value);
  Future<void> delete(String key);
}

/// Android Keystore destekli gerçek depo (`flutter_secure_storage`).
///
/// Veriler AES-GCM ile şifrelenir, anahtar Android Keystore'da tutulur.
/// Android otomatik yedeklemesi kapalıdır (AndroidManifest `allowBackup`):
/// başka cihaza geri yüklenen şifreli veri çözülemez.
class FlutterSecureStore implements SecureStore {
  FlutterSecureStore([FlutterSecureStorage? storage])
    : _storage = storage ?? const FlutterSecureStorage();

  final FlutterSecureStorage _storage;

  @override
  Future<String?> read(String key) => _storage.read(key: key);

  @override
  Future<void> write(String key, String value) =>
      _storage.write(key: key, value: value);

  @override
  Future<void> delete(String key) => _storage.delete(key: key);
}

/// Testler için bellek içi depo.
class InMemorySecureStore implements SecureStore {
  InMemorySecureStore([Map<String, String>? initial]) : values = {...?initial};

  final Map<String, String> values;

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async => values[key] = value;

  @override
  Future<void> delete(String key) async => values.remove(key);
}
