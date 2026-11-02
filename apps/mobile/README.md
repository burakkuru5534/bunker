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

Mobil mimari ve kurallar: [ARCHITECTURE.md](ARCHITECTURE.md). Genel mimari ve ekranlar: [../../docs/architecture.md](../../docs/architecture.md), [../../docs/ui/screen-map.md](../../docs/ui/screen-map.md).
