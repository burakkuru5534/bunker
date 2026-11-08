# Google Play Console — uygulama içeriği beyanları (SAFE-012)

Bu belge, Play Console → **Uygulama içeriği** bölümündeki formların cevaplarını uygulamanın **bugünkü** davranışından çıkarır. Uygulama değişince (ör. hata raporlama REL-004, reklam, ödeme, yeni veri) bu belge ve `legal/site/gizlilik.html` **aynı adımda** güncellenir.

Durum: taslak — formlar henüz doldurulmadı. Play'in güncel soru metinleri doldurma anında kontrol edilmelidir; aşağıdaki kategori adları Play yardım sayfasından alınmıştır (Kasım 2026).

## 1. Gizlilik politikası

- Adres: `https://burakkuru5534.github.io/siginak-legal/gizlilik.html` (yayın adımları `legal/README.md`).
- Sayfa uygulamanın Play'deki adını ("BUNKER — Son Kalanlar") ve veri sorumlusunu içerir.

## 2. Hesap silme

Uygulama misafir hesap açar (benzersiz, kalıcı kimlik) → Play'in hesap silme kuralı kapsamındadır.

| Soru | Cevap | Dayanak |
|---|---|---|
| Uygulama hesap oluşturuyor mu? | Evet — misafir hesap (kullanıcı adı/şifre yok, e-posta yok) | `POST /v1/sessions/guest` |
| Uygulama içi silme yolu | Ayarlar → **Hesabımı sil** (iki adımlı onay) | `DELETE /v1/me`, `settings_screen.dart` |
| Web'den silme bağlantısı | `…/gizlilik.html#hesap-silme` (bölümde e-posta ile talep yolu var) | Play: yeniden yükleme gerektirmeyen web yolu zorunlu |
| Silinen / tutulan veri | Oturumlar, sohbet mesajları, engellemeler, arkadaşlıklar, istekler ve davetler silinir; takma ad her yerde "Silinmiş oyuncu" olur; oyun sonuçları adsız kalır; raporlar güvenlik amacıyla belirtilen süre boyunca tutulur | `internal/account`, security.md §6 |

Erişimi kaybeden kullanıcı: Uygulamayı silen veya cihazını değiştiren misafir kullanıcı, e-postayla silme isteğinde hesabını kanıtlayamaz (hesapta e-posta yok; takma ad 2026-11-08'den beri benzersiz ama sahiplik kanıtı değil). Bu yüzden 90 gün kullanılmayan, oturumu kalmamış hesaplar bakım işinde otomatik anonimleştirilir (SAFE-014). Play de "toplanan veriyi 90 gün içinde otomatik silme/anonimleştirme"yi silme yolu olarak kabul eder. Gizlilik metninin silme bölümü e-posta talebini ve bu kuralı açıklar.

## 3. Veri güvenliği (Data safety) formu

### Genel sorular

| Soru | Cevap | Not |
|---|---|---|
| Uygulama gerekli veri türlerinden herhangi birini topluyor veya paylaşıyor mu? | Evet | |
| Toplanan tüm veriler aktarım sırasında şifreleniyor mu? | Evet | Üretim/staging derlemesinde `https`/`wss` zorunlu (`AppConfig`); Cloud Run yalnızca HTTPS sunar. Yerel `http` yalnızca geliştirme derlemesinde |
| Kullanıcılar verilerinin silinmesini isteyebilir mi? | Evet | Uygulama içi silme + web bağlantısı (§2) |
| Hesap oluşturma yöntemi | Diğer / misafir (kullanıcı adı-şifre, OAuth yok) | Formdaki seçenek adlarını doldururken kontrol edin |

### Paylaşım

**Hiçbir veri paylaşılmıyor.** Supabase (veritabanı) ve Google Cloud (sunucu) bizim adımıza ve talimatımızla veri işleyen **hizmet sağlayıcılardır**; Play tanımında bu aktarım "paylaşım" sayılmaz. Reklam, analitik veya üçüncü taraf SDK yok (bağımlılıklar: riverpod, go_router, dio, intl, flutter_secure_storage, url_launcher — hiçbiri veri göndermez).

### Toplanan veri türleri

| Play kategorisi → tür | Ne | Zorunlu / isteğe bağlı | Amaç |
|---|---|---|---|
| Kişisel bilgiler → **Ad** | Takma ad (Play tanımı takma adı da kapsar) | Zorunlu | Uygulama işlevi |
| Kişisel bilgiler → **Kullanıcı kimlikleri** | Rastgele üretilen kullanıcı kimliği (UUID) | Zorunlu | Uygulama işlevi; dolandırıcılık önleme, güvenlik |
| Mesajlar → **Diğer uygulama içi mesajlar** | Sohbet mesajları (30 gün); rapordaki son 50 mesajın kopyası | İsteğe bağlı (yazmadan oynanabilir) | Uygulama işlevi; dolandırıcılık önleme, güvenlik |
| Uygulama etkinliği → **Uygulama etkileşimleri** | Katılınan odalar, açılan kartlar, oylar, hazırım, oyun sonuçları | Zorunlu | Uygulama işlevi |
| Uygulama etkinliği → **Diğer kullanıcı içerikleri** | Rapor notu (isteğe bağlı, 200 karakter) | İsteğe bağlı | Dolandırıcılık önleme, güvenlik |
| Uygulama etkinliği → **Diğer işlemler** | Engellemeler, raporlar (kim, neden); arkadaş listesi, arkadaşlık istekleri, lobi davetleri | İsteğe bağlı | Uygulama işlevi; dolandırıcılık önleme, güvenlik |
| Cihaz veya diğer kimlikler → **Cihaz veya diğer kimlikler** | IP adresi (önerilen, temkinli beyan) | Zorunlu | Dolandırıcılık önleme, güvenlik |

IP için not: Sunucu IP'yi yalnızca bellekte, hesap açma sıklık sınırı için kullanır ve veritabanına yazmaz. Ancak Cloud Run istek günlükleri istemci IP'sini Google Cloud Logging'de tutar. Play'de IP'nin kesin kategorisi yok; IP'den konum çıkarılmadığı için "Konum" beyan edilmez. Temkinli yol olarak "Cihaz veya diğer kimlikler" altında beyan önerilir.

Beyan **edilmeyenler** (toplanmıyor): konum, e-posta, telefon, kişiler, fotoğraf/video, ses, dosyalar, takvim, finansal bilgi, sağlık, web geçmişi, yüklü uygulamalar, reklam kimliği, kilitlenme günlükleri ve teşhis verileri (REL-004 ile hata raporlama eklenirse "Uygulama bilgileri ve performansı → Kilitlenme günlükleri" eklenir).

"Sağlık bilgisi" uyarısı: Oyun kartlarındaki sağlık durumları (ör. "Şiddetli fıstık alerjisi") kurgusal karakter özellikleridir; kullanıcının kendi sağlık verisi değildir → beyan edilmez.

## 4. İçerik derecelendirmesi (IARC anketi)

Kategori: **Oyun** (sosyal / parti / kart). Anket cevapları, içerik kataloğuna (`migrations/postgres/0007_catalog_content.sql`) ve oyun tasarımına göre:

| Konu | Cevap | Dayanak |
|---|---|---|
| Şiddet | Yok — grafik şiddet yok; tema felaket sonrası hayatta kalma, eleme oylamayla ve sözlü | game-design.md |
| Korku / ürkütücü öğe | Yok (yalnızca metin; felaket senaryoları: nükleer kış, salgın, göktaşı, makineler, güneş fırtınası) | 0007 |
| Cinsellik / çıplaklık | Yok | 0007 |
| Kaba dil | Yok (uygulama içeriğinde). Kullanıcı sohbeti küfür filtresinden geçer | SAFE-013 |
| Uyuşturucu, alkol, tütün | Yok | 0007 (kart taraması) |
| Kumar / simüle kumar | Yok (kura yalnızca eşitlik bozma) | game-design.md |
| Kullanıcılar etkileşime giriyor mu? | **Evet** — oda içi yazılı sohbet (sesli/görüntülü yok) | PLAY-005 |
| Kullanıcı içeriği paylaşımı | Yalnızca odadaki oyuncularla yazılı sohbet | |
| Konum paylaşımı | Hayır | |
| Dijital satın alma | Hayır | product-requirements.md (ödeme yok) |
| Sınırsız internet erişimi | Hayır — yalnızca yasal metinleri harici tarayıcıda açar | `external_link.dart` |

Beklenen sonuç: düşük yaş derecesi ve "Kullanıcılar etkileşime giriyor" bilgisi. Kesin derece anketin sonucudur; burada tahmin edilmez.

## 5. Hedef kitle ve içerik

| Soru | Cevap | Not |
|---|---|---|
| Hedef yaş grupları | 13–15, 16–17, 18+ | Uygulama 13 yaş altına izin vermiyor (gizlilik §7, koşullar §2, hoş geldin notu) |
| Çocuklara çekici olabilir mi? | Hayır (soru sorulursa) | Aile programına (Families) başvurulmaz — 13 yaş altı hedef değil |
| Reklam içeriyor mu? | Hayır | |
| Reklam kimliği (AD_ID) | Kullanılmıyor | Manifest'te izin yok; reklam SDK'sı yok |
| Haber / sağlık / finans / devlet uygulaması | Hayır | |
| Uygulama erişimi (incelemeci için) | Özel giriş gerekmez; ancak oyun **en az 4 oyuncu** ister | Bkz. açık konu |

**Açık konu (REL-005/008):** Play incelemecisi tek başına oyunu başlatamaz. Seçenekler: "Uygulama erişimi" bölümüne açıklama (lobi, sohbet, raporla/engelle tek kişiyle denenebilir) ya da inceleme süresince sunucuya bağlanan test botları (`cmd/testbot`; bulutta çalıştırmak maliyet doğurabilir → karar o adımda sorulacak).

## 6. Kullanıcı içeriği (UGC) politikası kontrolü

Play, kullanıcı içeriği olan uygulamalardan aşağıdakileri ister:

| Gereklilik | Durum |
|---|---|
| Kullanıcı içerik oluşturmadan önce koşulları kabul eder | Hoş geldin ekranında "Başlayarak Kullanım Koşulları'nı kabul etmiş olursun" + bağlantı (SAFE-008) |
| Uygulama içi raporlama | Oyuncu menüsü → Raporla (5 neden + not) (SAFE-005) |
| Kullanıcı engelleme | Oyuncu menüsü → Engelle; engellenenler ekranı (SAFE-006) |
| Uygunsuz içeriğe karşı önlem | Küfür filtresi: takma ad reddi, sohbet maskeleme (SAFE-013); sıklık sınırları |
| Raporların incelenmesi | Raporlar veritabanında; **inceleme aracı/süreci henüz yok** (yönetim paneli planlanmadı) → yayından önce en az SQL ile inceleme rutini belgelenmeli (REL-008) |
