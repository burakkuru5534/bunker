# ADR-0003: Tek instance Cloud Run ve bellek içi WebSocket hub

- **Durum:** Kabul edildi (2026-11-02)

## Bağlam
Gerçek zamanlı güncellemeler WebSocket ile gönderilecek. Redis ve ayrı mesajlaşma servisi yasak. Bir odadaki tüm oyuncuların aynı yayını alması gerekiyor.

## Karar
- Cloud Run servisi **max-instances = 1** ile çalışır.
- Bağlantılar bellekte `oda → bağlantılar` eşlemesinde tutulur.
- Her bağlantının yazma goroutine'i ve sınırlı gönderim kuyruğu vardır; kuyruğu dolan istemci kesilir.

## Alternatifler
- **Çoklu instance + Redis pub/sub:** Redis yasak.
- **Çoklu instance + PostgreSQL LISTEN/NOTIFY:** Mümkün ama MVP için gereksiz karmaşıklık; Supabase pooler modlarıyla uyumu ayrıca doğrulanmalı. Ölçek ihtiyacı doğarsa ilk aday.
- **Cloud Run session affinity ile çoklu instance:** Aynı odadaki oyuncuların aynı instance'a düşmesini garanti etmez.

## Sonuçlar
- Basit ve ucuz; koordinasyon kodu yok.
- Tek instance kapasitesi üst sınırdır (RT-011'de ölçülecek).
- Deploy ve instance değişiminde tüm bağlantılar kopar; istemci yeniden bağlanmayı sorunsuz yönetmelidir (ADR-0006).
