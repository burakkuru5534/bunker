# ADR-0002: Modüler monolith ve saf oyun motoru

- **Durum:** Kabul edildi (2026-11-02)

## Bağlam
MVP küçük ölçekli, tek ekip, düşük maliyet hedefli. Oyun kuralları (aşamalar, oylama, eşitlik bozma, kazanma) karmaşık ve hataya açık; kapsamlı test edilmesi gerekiyor.

## Karar
- Tek Go servisi; `internal/` altında modüller: config, observability, httpapi, auth, room, game (engine/service/timers), cards, chat, moderation, realtime, persistence.
- Katmanlar: transport → service → engine. Handler'larda iş kuralı yok.
- `game/engine` saf fonksiyonlardan oluşur: `(durum, eylem, zaman, rng) → (yeni durum, olaylar, hata)`. I/O, sistem saati ve global rastgelelik yoktur.
- Master prompt'tan sapmalar: `lobby`→`room`, `voting` motorun içinde, `chat` modülü eklendi, entegrasyon testleri `backend/tests/integration`.

## Alternatifler
- **Mikroservisler:** Proje kuralı gereği yasak; MVP için gereksiz operasyon yükü.
- **Kuralların service/handler katmanında olması:** Test için veritabanı gerektirir, kurallar dağılır.

## Sonuçlar
- Oyun kuralları milisaniyeler içinde, deterministik birim testleriyle doğrulanır.
- Modüller arası bağımlılık tek yönlü tutulmalı; engine hiçbir modüle bağımlı olmamalıdır.
