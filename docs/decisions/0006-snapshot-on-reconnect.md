# ADR-0006: Yeniden bağlanmada tam durum görüntüsü

- **Durum:** Kabul edildi (2026-11-02)

## Bağlam
Mobil ağlar kopar, uygulama arka plana alınır, deploy'larda tüm WS bağlantıları düşer. Yeniden bağlanan oyuncu doğru duruma dönmelidir.

## Karar
- İstemci yeniden bağlandığında bildiği son `state_version`'ı gönderir.
- Sunucu, oyuncuya özel **tam durum görüntüsünü** (açılmış kartlar, oyuncunun kendi kartları, aşama ve deadline, sayaçlar) ve son 50 sohbet mesajını gönderir.
- Sonraki olaylar `state_version` ile gelir; istemci sürüm atlaması görürse tam görüntü ister.

## Alternatifler
- **Olay günlüğünden kaçırılan olayları yeniden oynatma:** Daha az veri ama olay saklama, sıralama ve boşluk yönetimi gerektirir; hataya açık.

## Sonuçlar
- İstemci mantığı basit: "her zaman en son görüntüyü göster".
- Görüntü boyutu küçük (≤10 oyuncu × 5 kart); maliyet etkisi önemsiz.
- Gizli bilgi izolasyonu görüntü üretiminde de test edilmelidir (ENG-015).
