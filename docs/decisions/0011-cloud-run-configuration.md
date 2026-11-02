# ADR-0011: Cloud Run yapılandırması — instance-based, min 0 / max 1, europe-west1

- **Durum:** Kabul edildi (2026-11-02)

## Bağlam
Backend WebSocket bağlantıları ve bellek içi zamanlayıcılar kullanır (ADR-0003, ADR-0005). Cloud Run'da açık WS bağlantısı olan instance aktif sayılır ve instance-based faturalanır. Maliyet düşük ve öngörülebilir olmalı.

## Karar
- Bölge `europe-west1` (Tier 1). Servisler: `bunker-api-test`, `bunker-api`.
- **Instance-based faturalama** (CPU her zaman tahsisli).
- min-instances **0**, max-instances **1**, 1 vCPU / 512 MiB, eşzamanlılık 1000, istek zaman aşımı 3600 sn, session affinity açık.
- Startup probe `/health`; SIGTERM'de WS'ler `1012` ile kapatılır.
- Ortam başına ayrı servis hesabı; yalnızca kendi secret'ına `secretAccessor`.

## Alternatifler
- **Request-based faturalama:** İstek yokken CPU kısıtlanır; zamanlayıcı goroutine'leri güvenilir çalışmaz. WS açıkken zaten instance-based faturalanır, fiyat avantajı kalmaz.
- **min-instances=1:** Soğuk başlatma olmaz ama 7/24 ücret (≈ $44/ay, Tier 1). Beta sonrası gerekirse yeniden değerlendirilir.
- **europe-west3 (Frankfurt):** Supabase'e daha yakın ama Tier 2 fiyat.

## Sonuçlar
- Boşta ücret yok; aylık tavan tek instance'ın 7/24 maliyeti.
- Bağlantı yokken instance kapanabilir; aktif oyunların deadline'ları bir sonraki açılışta işlenir.
- Cloud Run (Belçika) ↔ Supabase (Frankfurt) arasında birkaç ms gecikme; oyun için önemsiz.
