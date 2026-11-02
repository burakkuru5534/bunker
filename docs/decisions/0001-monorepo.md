# ADR-0001: Monorepo

- **Durum:** Kabul edildi (2026-11-02)

## Bağlam
Flutter uygulaması, Go backend, migration'lar ve dokümanlar birlikte gelişecek. API ve WS sözleşmeleri istemci ile sunucu arasında sık değişecek. Tek geliştirici çalışıyor.

## Karar
Tek repository: `apps/mobile`, `backend`, `migrations/postgres`, `docs`, `scripts`.

## Alternatifler
- **Ayrı repolar (mobile / backend):** Bağımsız sürümleme sağlar ama sözleşme değişikliklerinde iki repoda eşzamanlı PR gerekir; tek kişi için gereksiz yük.

## Sonuçlar
- Bir sözleşme değişikliği istemci ve sunucuyla tek commit'te yapılabilir.
- CI, değişen klasöre göre (path filter) yalnızca ilgili kontrolleri çalıştırmalıdır.
