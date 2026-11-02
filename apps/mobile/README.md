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
flutter run            # bağlı cihaz veya emülatör
```

## Kontroller

```bash
dart format --set-exit-if-changed .
flutter analyze
flutter test
flutter build apk --debug
```

Mimari ve ekranlar: [../../docs/architecture.md](../../docs/architecture.md), [../../docs/ui/screen-map.md](../../docs/ui/screen-map.md).
