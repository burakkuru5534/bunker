# Sığınak: Son Kalanlar

Türkiye pazarına yönelik, çok oyunculu, çevrim içi sosyal tartışma ve eleme oyunu.
Dünyayı bir felaket vurmuştur; oyuncular kapasitesi sınırlı bir sığınağa girebilmek için
kartlarını açar, tartışır ve gizli oyla birbirlerini eler.

## Durum

Proje **Faz 0 — Keşif ve ürün tanımı** aşamasındadır. Henüz kod yoktur.
Görevlerin güncel durumu: [docs/roadmap.md](docs/roadmap.md)

## Teknoloji

| Katman | Teknoloji |
|---|---|
| Mobil | Flutter (Android öncelikli), Riverpod, go_router, Dio, WebSocket |
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
- [Mimari kararlar (ADR)](docs/decisions/README.md)
- [Yol haritası ve görevler](docs/roadmap.md)

## Geliştirme ortamı

| Araç | Sürüm |
|---|---|
| Flutter | 3.47.1 (stable) |
| Dart | 3.13.1 |
| Go | 1.27.0 |
| Android SDK | 36.0.0 |

Yerel geliştirme, yapılandırma ve test talimatları ilgili görevlerde eklenecektir.

> Gizli değerler (Supabase, Google Cloud anahtarları) asla repoya eklenmez. Yerel değerler `.env` dosyasında tutulur ve `.gitignore` ile dışlanır.
