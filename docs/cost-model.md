# Maliyet Modeli — Sığınak: Son Kalanlar

> Kaynak: ARCH-003 (2026-11-02). Fiyatlar resmi sayfalardan alınmıştır (Cloud Run ve ağ: **2026-11-05**'te yeniden doğrulandı); **değişebilir**, her ücretli adımdan önce yeniden doğrulanır.
> Ücret doğurabilecek her kaynak değişikliği için **kullanıcı onayı** gerekir.

## 1. Kaynaklar

- Cloud Run fiyatlandırması: https://cloud.google.com/run/pricing
- Cloud Run WebSocket davranışı: https://docs.cloud.google.com/run/docs/triggering/websockets
- Supabase fiyatlandırması: https://supabase.com/pricing
- Ağ (internet çıkışı) fiyatlandırması: https://cloud.google.com/vpc/network-pricing

## 2. Cloud Run — önemli kurallar

- **Açık WebSocket bağlantısı olan instance aktif sayılır**; CPU tahsis edilir ve **instance-based** faturalanır. Boşta kalan bağlantılar da ücretlidir.
- WebSocket bir HTTP isteğidir; istek zaman aşımı en fazla **60 dk** (varsayılan 5 dk). İstemci periyodik kopmaları otomatik yönetmelidir.
- Instance başına en fazla **1.000 eşzamanlı bağlantı** → max-instances=1 ile MVP üst sınırı ≈ 1.000 eşzamanlı oyuncu.
- Bağlantı yoksa ve min-instances=0 ise **ücret yok**.
- Instance-based faturalamada instance'ın **tüm ömrü** ücretlidir (başlatmadan kapanmaya, en az 1 dk). Son bağlantı kapandıktan sonra Cloud Run boştaki instance'ı bir süre (yaklaşık 15 dk'ya kadar) tutabilir; bu kuyruk da ücretlidir.
- **En büyük maliyet sürücüsü açık kalan WebSocket'tir:** tek bir oyuncunun uygulaması arka planda odada açık kalırsa instance aktif kalır. Mobil istemci şu an arka plana geçince bağlantıyı kapatmıyor → düzeltme önerildi (REL-003 öncesi).

### Fiyatlar (varsayılan / Tier 1, USD)

| Kalem | Request-based | Instance-based |
|---|---|---|
| CPU | $0.000024 / vCPU-sn | $0.000018 / vCPU-sn |
| Bellek | $0.0000025 / GiB-sn | $0.000002 / GiB-sn |
| İstek | $0.40 / milyon | — |
| Aylık ücretsiz (faturalama hesabı başına) | 180.000 vCPU-sn, 360.000 GiB-sn, 2 milyon istek | 240.000 vCPU-sn, 450.000 GiB-sn |

- `europe-west1` (Belçika) **Tier 1**, `europe-west3` (Frankfurt) **Tier 2** (daha pahalı). → Bölge seçimi ARCH-005'te maliyet açısından değerlendirilecek.
- Ücretsiz kotalar faturalama hesabı geneline uygulanır; aynı hesaptaki diğer projeler (varsa) kotayı paylaşır.

## 3. Supabase

| Plan | Fiyat | Notlar |
|---|---|---|
| Free | $0 | Proje başına 500 MB veritabanı, 5 GB egress, paylaşımlı CPU / 500 MB RAM, **en fazla 2 aktif proje**, **1 hafta hareketsizlikte proje duraklatılır** |
| Pro | $25 / ay'dan başlar | İlk proje dahil, $10 compute kredisi (1 Micro instance'ı karşılar); ek proje $10 / ay'dan; harcama limiti (spend cap) varsayılan açık |
| Micro compute | $10 / ay | 1 GB RAM, 60 doğrudan bağlantı, 200 pooler bağlantısı |

## 4. Senaryolar (2026-11-05; Tier 1, 1 vCPU / 512 MiB, instance-based, min 0 / max 1)

Birim: instance'ın açık kaldığı her saat **CPU $0,0648 + bellek $0,0036**. Ücretsiz kota (faturalama hesabı başına, ayda): 240.000 vCPU-sn ≈ **66,7 saat** CPU, 450.000 GiB-sn ≈ **250 saat** bellek (512 MiB'de). Test ve prod servisleri ve aynı faturalama hesabındaki diğer Cloud Run kullanımları bu kotayı **paylaşır**.

| Senaryo | Instance açık (saat/ay) | Hesap | Cloud Run / ay |
|---|---|---|---|
| Geliştirme ve test | ~30 | Kota içinde | **$0** |
| Kapalı beta (haftada 3 akşam × ~3 saat + boşta kuyruklar) | ~50 | Kota içinde | **$0** |
| Yoğun beta / küçük yayın (günde ~4 saat) | ~120 | (120 − 66,7) × 0,0648 | **≈ $3,5** |
| Günde ~12 saat oyun | ~360 | 293,3 × 0,0648 + 110 × 0,0036 | **≈ $19,4** |
| Tavan: 7/24 açık (max-instances=1) | 720 | 653,3 × 0,0648 + 470 × 0,0036 | **≈ $44,0** |

Ek kalemler:

| Kalem | Tahmin | Not |
|---|---|---|
| İnternet çıkışı (WS → oyuncular, sunucu → Supabase) | Maç başına ~2–5 MB; 500 maç/ay ≈ 1–2,5 GiB → **$0–0,2**; 5.000 maç/ay ≈ 10–25 GiB → **$1,2–3** | Premium Tier, Avrupa: ilk 1 GiB ücretsiz, sonra $0,12/GiB. Maç başı hacim QA-007'de ölçülecek |
| Cloud Logging | **$0** | Aylık 50 GiB ücretsiz (security.md §7) |
| Artifact Registry (imaj), Cloud Build | **$0 beklenir** | Küçük distroless imaj; kotalar dağıtım adımında (REL-003) doğrulanır |
| Secret Manager | **Kullanılmıyor** | Karar 2026-11-05 (security.md §3) |
| Supabase | Free: **$0** | Free proje 1 hafta hareketsizlikte duraklar; herkese açık yayında Pro ($25/ay) ayrıca değerlendirilir |

- **Tavan:** max-instances=1 olduğu için Cloud Run'ın aylık üst sınırı ≈ $44 + çıkış trafiği. Bunu aşmak için önce max-instances'ın değiştirilmesi gerekir (onay gerektirir).
- Kur farkı nedeniyle tutarlar USD'dir; fatura TL kesilebilir.

## 5. Maliyet kontrolleri

- Cloud Run: min-instances=0, max-instances=1.
- Google Cloud **bütçe uyarısı** (ör. $5 ve $20) — ücretsiz; kullanıcı panelden kurar.
- Supabase Pro'ya geçilirse spend cap açık kalır.
- Gereksiz polling yok; WS yalnızca ilgili odaya yayın.
- Loglar: bilgi seviyesi kısa; sohbet içeriği ve istek gövdeleri loglanmaz.
- Ölçülecek metrikler (QA-007, RT-012): instance aktif süresi, eşzamanlı WS, maç başına DB yazma (~600 tahmini), oda başına mesaj, log hacmi.

## 6. Açık maliyet kararları

| Karar | Ne zaman | Durum |
|---|---|---|
| Supabase planı | Şimdilik **Free**, tek proje (test + prod şemaları, ADR-0009). Pro'ya geçiş yayından önce yeniden değerlendirilecek. | Karar verildi (2026-11-02) |
| Cloud Run bölgesi | **europe-west1** (Tier 1); Supabase **eu-central-1** | Karar verildi (2026-11-02) |
| Supabase bağlantı yolu | Supavisor session mode (IPv4) — ücretli IPv4 eklentisine gerek yok (ADR-0010) | Karar verildi (2026-11-02) |
| Cloud Run faturalama | Instance-based, min 0 / max 1 (ADR-0011) | Karar verildi (2026-11-02) |
| GCP bütçe uyarısı | Bulut kaynakları oluşturulmadan önce | Bekliyor — kullanıcı işlemi |
| Secret Manager | `DATABASE_URL` Cloud Run ortam değişkeni olarak verilir | Karar verildi (2026-11-05): **kullanılmaz** |
