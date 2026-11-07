# Sığınak — Mobil uygulama (Flutter)

| | |
|---|---|
| Paket adı (Dart) | `siginak` |
| Android applicationId | `com.bunker.app` |
| Platform | Android (iOS MVP sonrası) |
| Flutter / Dart | 3.47.1 / 3.13.1 |

## Çalıştırma

```bash
cd apps/mobile
flutter pub get
cp config/local.example.json config/local.json   # ilk kez; gerekirse IP'yi düzenle
flutter run --dart-define-from-file=config/local.json
```

Ortamlar (`local`, `test`, `prod`) ve adres kuralları: [config/README.md](config/README.md).

## Kontroller

```bash
dart format --set-exit-if-changed .
flutter analyze
flutter test
flutter build apk --debug
```

## Release imzası ve Play paketi

Release derlemeleri **yükleme anahtarıyla** imzalanır (Play App Signing açık; uygulamanın asıl imza anahtarını Google tutar). Anahtar ve şifreler git'e girmez (`android/.gitignore`: `key.properties`, `**/*.jks`):

| Dosya | İçerik |
|---|---|
| `android/keystore/upload-keystore.jks` | Yükleme anahtarı (PKCS12, RSA 4096, takma ad `upload`, 2054'e kadar geçerli) |
| `android/key.properties` | `storePassword`, `keyPassword`, `keyAlias=upload`, `storeFile=../keystore/upload-keystore.jks` |

- `key.properties` yoksa release derlemesi açık bir hatayla durur (debug anahtarıyla sessizce imzalanmaz). Debug derlemeleri ve CI etkilenmez.
- **İkisini de repo dışına yedekleyin** (şifre yöneticisi / güvenli bulut klasörü). Kaybolursa Play Console'dan yükleme anahtarı sıfırlama istenebilir, ancak bu günler sürer.
- Sertifika parmak izi: `keytool -list -v -keystore android/keystore/upload-keystore.jks -alias upload`.

```bash
# Play'e yüklenecek paket (sürüm pubspec.yaml'dan: 1.0.0+1)
flutter build appbundle --release --dart-define-from-file=config/prod.json
# Çıktı: build/app/outputs/bundle/release/app-release.aab
```

Her yüklemede `pubspec.yaml`'daki `+` sonrası sayı (versionCode) artırılır; `+` öncesi değişirse `lib/core/config/app_version.dart` de güncellenir (test denetler).

Mobil mimari ve kurallar: [ARCHITECTURE.md](ARCHITECTURE.md). Genel mimari ve ekranlar: [../../docs/architecture.md](../../docs/architecture.md), [../../docs/ui/screen-map.md](../../docs/ui/screen-map.md).
