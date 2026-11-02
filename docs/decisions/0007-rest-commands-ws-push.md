# ADR-0007: Komutlar REST ile, WebSocket yalnızca sunucu → istemci

- **Durum:** Kabul edildi (2026-11-02)

## Bağlam
Oyuncu eylemleri (kart açma, hazırım, oy, sohbet) sunucuya iletilmeli; durum değişiklikleri tüm odaya gerçek zamanlı yayılmalı. Eylemlerin doğrulanması, yetkilendirilmesi, sıklık sınırı ve idempotency'si gerekiyor.

## Karar
- Tüm komutlar REST (`POST`) ile, istek gövdesinde `action_id` / `client_msg_id` ile gönderilir.
- WebSocket yalnızca sunucudan istemciye olay ve durum taşır; istemciden yalnızca `sync` mesajı kabul edilir.
- İstemci UI'ı WS olaylarından günceller; HTTP yanıtı yalnızca onay/hatadır.

## Alternatifler
- **Çift yönlü WebSocket komutları:** Biraz daha düşük gecikme; ancak onay/hata eşleştirme, ayrı mesaj doğrulama ve ayrı sıklık sınırı gerektirir, iki komut yolu oluşur.

## Sonuçlar
- Tek doğrulama/yetki/sıklık sınırı yolu (HTTP middleware), kolay entegrasyon testi.
- WS sunucusu basit kalır.
- Her eylem ayrı HTTP isteğidir; bu ölçekte gecikme ve Cloud Run maliyet etkisi önemsizdir (RT-012'de ölçülecek).
