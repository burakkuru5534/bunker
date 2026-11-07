# ADR-0013: Sıfır maliyet — backend Render Free'de, ödeme yöntemi eklenmez

- **Durum:** Uygulanmadı (2026-11-07). Kullanıcı en düşük maliyetli Cloud Run kurulumunu seçti; geçerli karar [ADR-0011](0011-cloud-run-configuration.md). Bu belge değerlendirme kaydı olarak tutulur.
- ~~Yerini aldığı: ADR-0011~~

## Bağlam
Hedef **sıfır maliyet**. Cloud Run'ın ücretsiz kotası (instance-based, ayda ~67 saat CPU) aşılınca ücret çıkar ve Google Cloud'da harcamayı durduran bir tavan yoktur; bütçe uyarısı yalnızca haber verir (cost-model.md). Compute Engine e2-micro yalnızca ABD bölgelerinde ücretsizdir ve IPv4 adresi ücretlidir (~3,65 $/ay). Oracle Always Free VM ücretsizdir ama kart doğrulaması ve sunucu bakımı ister.

## Karar
- Backend, **Render Free web service** (Docker imajı, plan `free`, 512 MB RAM) olarak çalışır; tek servis (`bunker-api`). Bölge: Frankfurt (Supabase `eu-central-1`'e yakın; kurulumda ücretsiz planda seçilebildiği doğrulanır).
- Render çalışma alanı **Hobby (0 $)**; **ödeme yöntemi eklenmez.** Kota aşılırsa fatura çıkmaz, Free servisler ayın kalanında askıya alınır. Bu, maliyet için sert tavandır (kesinti riskiyle).
- Kotalar (Render, 2026-11-06): çalışma alanı başına ayda **750 instance saati**, **5 GB dışarı trafik** (1 Ağustos 2026'da 100 GB'tan düşürüldü), **500 derleme dakikası**.
- Test ortamı için ikinci servis yalnızca gerektiğinde açılır ve iş bitince silinir/durdurulur (750 saat paylaşılır; iki servis 7/24 sığmaz).
- `DATABASE_URL` Render ortam değişkeni olarak verilir (Secret Manager yok; security.md §3). Çalışma alanına başka üye eklenmez.
- Veritabanı **Supabase Free**'de kalır. Bir hafta hareketsizlikte duraklamasın diye GitHub Actions zamanlanmış işi günde bir kez `GET /health/ready` çağırır (sunucuyu uyandırır, veritabanına sorgu gider).

## Uyku ve yeniden başlatma
- Free servis **15 dk gelen trafik yoksa uyur**; ilk istek onu ~1 dk'da uyandırır. WebSocket mesajları trafik sayılır.
- Odadaki oyuncular varken servis uyumamalı: istemci odadayken periyodik hafif bir WS mesajı gönderir (yapılacak). Mobil uygulama ilk istekte uzun bekleme için "sunucu uyanıyor" durumunu gösterir (yapılacak).
- Render servisi istediği an yeniden başlatabilir. Mimari buna dayanıklıdır: durum veritabanında (ADR-0004), süreler kalıcı (ADR-0005), yeniden bağlanan istemci anlık görüntü alır (ADR-0006). Tek instance ve bellek içi hub (ADR-0003) aynen geçerlidir (Free plan tek instance).

## Trafik bütçesi
- 5 GB/ay ≈ maç başına 2–5 MB tahminiyle **1.000–2.500 maç/ay** (ölçülecek, QA-007). Yaklaşınca seçenekler: WS sıkıştırma (`permessage-deflate`), mesaj boyutlarının küçültülmesi; yetmezse ücretli plana geçiş (onayla).

## Alternatifler
- **Cloud Run, kota içinde (ADR-0011):** Hızlı uyanma, ama kart ve faturalama hesabı gerekir, kota aşılırsa ücret (tavan ~44 $/ay). Trafik büyürse yeniden değerlendirilir; yapılandırma hazır.
- **Oracle Always Free VM:** 7/24 ve güçlü; kart doğrulaması, TLS ve bakım sorumluluğu, boşta kalan ücretsiz sunucuların geri alınması riski.
- **Compute Engine e2-micro:** ABD bölgesi (gecikme) ve ücretli IPv4.

## Sonuçlar
- Maliyet 0 $; bedeli: uyku sonrası ~1 dk soğuk başlangıç, aylık kota aşımında kesinti, Render'ın "üretim için değil" uyarısı.
- Cloud Logging yerine Render log ekranı kullanılır (güvenlik olayı alanları aynı; saklama süresi kurulumda doğrulanır).
- `TRUSTED_PROXY_HOPS` Render'ın vekil zinciri için kurulumda doğrulanır.
