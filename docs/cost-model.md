# Maliyet Modeli — Sığınak: Son Kalanlar

> Kaynak: ARCH-003 (2026-11-02). Fiyatlar resmi sayfalardan 2026-11-02 tarihinde alınmıştır; **değişebilir**, her ücretli adımdan önce yeniden doğrulanır.
> Ücret doğurabilecek her kaynak değişikliği için **kullanıcı onayı** gerekir.

## 1. Kaynaklar

- Cloud Run fiyatlandırması: https://cloud.google.com/run/pricing
- Cloud Run WebSocket davranışı: https://docs.cloud.google.com/run/docs/triggering/websockets
- Supabase fiyatlandırması: https://supabase.com/pricing

## 2. Cloud Run — önemli kurallar

- **Açık WebSocket bağlantısı olan instance aktif sayılır**; CPU tahsis edilir ve **instance-based** faturalanır. Boşta kalan bağlantılar da ücretlidir.
- WebSocket bir HTTP isteğidir; istek zaman aşımı en fazla **60 dk** (varsayılan 5 dk). İstemci periyodik kopmaları otomatik yönetmelidir.
- Instance başına en fazla **1.000 eşzamanlı bağlantı** → max-instances=1 ile MVP üst sınırı ≈ 1.000 eşzamanlı oyuncu.
- Bağlantı yoksa ve min-instances=0 ise **ücret yok**.

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

## 4. Senaryolar (tahmini, Tier 1, 1 vCPU / 512 MiB, max-instances=1)

| Senaryo | Varsayım | Cloud Run | Supabase | Toplam / ay |
|---|---|---|---|---|
| Geliştirme | Aylık ~20 saat aktif bağlantı | $0 (ücretsiz kota içinde) | Free (tek proje, test + prod şemaları) | **$0** |
| Kapalı beta | Aylık ~200 saat aktif, ~1 milyon istek | ≈ $9 (CPU; bellek ve istek ücretsiz kota içinde) | Free | **≈ $9** |

> Not: Free projede test ve prod aynı kaynakları paylaşır; proje 1 hafta hareketsiz kalırsa duraklar (kapalı beta sırasında risk).
| Tavan | Instance 7/24 aktif (720 saat) | ≈ $44 (CPU ≈ $42 + bellek ≈ $2) | Pro: $25 | **≈ $69** |

Hesap (tavan, CPU): `720 × 3.600 = 2.592.000 vCPU-sn − 240.000 ücretsiz = 2.352.000 × $0.000018 ≈ $42`.

- **max-instances=1**, Cloud Run maliyetine doğal bir tavan koyar (tek instance'ın 7/24 maliyeti).
- Tier 2 bölge seçilirse Cloud Run kalemleri daha yüksek olur.
- Egress (WS mesajları küçük), Secret Manager, Artifact Registry ve Cloud Logging bu ölçekte ücretsiz kotalar içinde kalması beklenir; ilgili görevlerde (ARCH-005, REL-003) güncel kotalar doğrulanacak.

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
