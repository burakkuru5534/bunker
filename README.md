# BUNKER — Son Kalanlar

Türkiye pazarına yönelik, çok oyunculu, çevrim içi sosyal tartışma ve eleme oyunu.
Dünyayı bir felaket vurmuştur; oyuncular kapasitesi sınırlı bir sığınağa girebilmek için
kartlarını açar, tartışır ve gizli oyla birbirlerini eler.

## Durum

Proje **Faz 0 — Keşif ve ürün tanımı** aşamasındadır. Henüz kod yoktur.
Görevlerin güncel durumu: [docs/roadmap.md](docs/roadmap.md)

## Teknoloji

| Katman | Teknoloji |
|---|---|
| Mobil | Flutter (Android öncelikli; `com.bunker.app`), Riverpod, go_router, Dio, WebSocket — [apps/mobile](apps/mobile/README.md) |
| Backend | Go modüler monolith, REST + WebSocket |
| Veritabanı | Supabase PostgreSQL (tek kalıcı veri kaynağı) |
| Dağıtım | Google Cloud Run |
| Kimlik | Backend tarafından üretilen misafir oturumu (Firebase yok) |

## Planlanan repository yapısı

```
bunker/
  apps/mobile/     Flutter uygulaması
  backend/         Go servisi (cmd/server, internal/...)
  migrations/      PostgreSQL migration'ları
  docs/            Ürün, oyun tasarımı, mimari dokümanları
  scripts/         Yardımcı betikler
```

## Dokümanlar

- [Ürün gereksinimleri](docs/product-requirements.md)
- [Oyun tasarımı](docs/game-design.md)
- [Sistem mimarisi](docs/architecture.md)
- [Veri modeli (taslak)](docs/data-model.md)
- [API sözleşmeleri](docs/api.md)
- [Güvenlik](docs/security.md)
- [Maliyet modeli](docs/cost-model.md)
- [Mimari kararlar (ADR)](docs/decisions/README.md)
- [Marka ve görsel dil](docs/ui/brand.md)
- [Ekran haritası](docs/ui/screen-map.md)
- [Tasarım sistemi](docs/ui/design-system.md)
- [Yol haritası ve görevler](docs/roadmap.md)

## CI

GitHub Actions — [.github/workflows/mobile.yml](.github/workflows/mobile.yml): mobil dosyalar değişince format, l10n, analiz ve testler; `main`'e push'ta Android debug derlemesi. Yerelde aynı kontrol:

```bash
cd apps/mobile
dart format --set-exit-if-changed . && flutter analyze --fatal-infos && flutter test
```

## Geliştirme ortamı

| Araç | Sürüm |
|---|---|
| Flutter | 3.47.1 (stable) |
| Dart | 3.13.1 |
| Go | 1.27.3 |
| Android SDK | 36.0.0 |

Yerel geliştirme, yapılandırma ve test talimatları ilgili görevlerde eklenecektir.

> Gizli değerler (Supabase, Google Cloud anahtarları) asla repoya eklenmez. Yerel değerler `.env` dosyasında tutulur ve `.gitignore` ile dışlanır.
