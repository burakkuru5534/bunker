# Yayın kontrol listesi (REL-008)

Google Play (Android) ilk yayın için. Durum: ✅ hazır · 🟡 kısmen · ⬜ yapılmadı · ❓ karar bekliyor. Güncelleme: 2026-11-07.

## A. Yayını engelleyenler

| # | Madde | Durum | Not |
|---|---|---|---|
| A1 | Sunucu barındırma kararı | ✅ | Cloud Run, en düşük maliyet (ADR-0011; cost-model.md §0). Render Free uygulanmadı (ADR-0013) |
| A2 | Sunucu dağıtımı | ✅ | `backend/Dockerfile` ✅, test servisi `bunker-api-test` ✅ (europe-west1; DB bağlı, duman testi ve 4 botla oyun geçti; `config/test.json` gerçek adreste). Prod `bunker-api` ✅ (2026-11-07; DB bağlı, duman testi geçti). Yayında `MIN_APP_VERSION` sürüm politikasına göre ayarlanır (`gcloud run services update bunker-api --update-env-vars MIN_APP_VERSION=…`) |
| A3 | Prod veritabanı | ✅ | `bunker_prod` şeması: `bunker_prod_app` / `bunker_prod_migrator` rolleri (`dbsetup`), migration 0001–0008 uygulandı (2026-11-07, kullanıcı) |
| A4 | Mobil prod adresleri | ✅ | `apps/mobile/config/prod.json` → `https://bunker-api-918364240578.europe-west1.run.app` (wss aynı) |
| A5 | Uygulama adı | ❓ | Logo "BUNKER — Son Kalanlar"; uygulama etiketi, `appTitle`, yasal sayfalar "Sığınak". Tek ad seçilip hepsi eşitlenir (Play adı ≤ 30 karakter) |
| A6 | Release imzası | ✅ | Yükleme anahtarı `android/keystore/upload-keystore.jks` (PKCS12, RSA 4096, takma ad `upload`, 2054'e kadar) + `android/key.properties` (git dışı); `signingConfigs.release`. `key.properties` yoksa release derlemesi hatayla durur. Play App Signing ilk yüklemede açılır. **Anahtar ve şifre repo dışına yedeklenmeli** |
| A7 | Sürüm ve paket | 🟡 | Sürüm 1.0.0+1 ✅; imza yapılandırması geçici anahtarla derlenen AAB'de doğrulandı ✅. Kalan: gerçek anahtarla `flutter build appbundle --release --dart-define-from-file=config/prod.json` (kullanıcı bilgisayarında) ve Play'e yükleme. Hedef API 36 ✅ |
| A8 | Yasal sayfalar | 🟡 | Metinler hazır (`legal/site`); GitHub Pages'te yayın ve adreslerin açıldığının kontrolü |
| A9 | Simge ve açılış ekranı | ✅ | `branding/` (kaynak + üretim betiği), uyarlanabilir simge, Android 12+ açılış, uygulama içi logo; Play simgesi `branding/play_icon_512.png` |
| A10 | Mağaza kaydı | ⬜ | Kısa açıklama (≤ 80), tam açıklama (≤ 4000), **tanıtım görseli 1024×500**, **en az 2 telefon ekran görüntüsü**, kategori (Oyun), iletişim e-postası, gizlilik adresi |
| A11 | Uygulama içeriği formları | 🟡 | Cevaplar `docs/store/play-console.md`'de (Cloud Run/Cloud Logging IP notu dahil). "Uygulama erişimi"ne inceleme notu: oyun 4 oyuncu ister |
| A12 | Kapalı test | ❓ | Kişisel geliştirici hesabı 13 Kasım 2023'ten sonra açıldıysa: en az **12 test kullanıcısı, 14 gün kesintisiz**, ardından üretim erişimi başvurusu |
| A13 | Arka planda bağlantı | ✅ | Odadayken uygulama 30 sn arka planda kalırsa WS normal kodla (1000) kapatılır (`RealtimeStatus.suspended`), öne gelince aynı odaya bağlanıp snapshot ile eşitlenir; kısa geçişler (kodu paylaşmak) bağlantıyı düşürmez. Cloud Run boşta instance saati harcamaz |

## B. Yayından önce yapılması gerekenler

| # | Madde | Durum | Not |
|---|---|---|---|
| B1 | Gerçek cihazlarla uçtan uca oyun | ⬜ | En az 4 oyuncu (telefon/emülatör + `cmd/testbot`) dağıtılmış sunucuya karşı; kopma/yeniden bağlanma, arka plan, eleme, maç sonu |
| B2 | Supabase gizli bilgileri | ⬜ | Sohbete yazılmış `postgres` şifresi ve `SUPABASE_SECRET_KEY` yenilenir (security.md §3) |
| B3 | Supabase Free duraklama | ⬜ | Bir hafta hareketsizlikte duraklar; günlük GitHub Actions dürtmesi (`/health/ready`) |
| B4 | Rapor inceleme rutini | ⬜ | Panel yok; raporları SQL ile inceleme ve sonuçlandırma adımları belgelenir |
| B5 | Hata takibi (REL-004) | ⬜ | Ücretsiz seçenek (ör. Sentry ücretsiz plan) ya da ilk sürümde yok; Firebase kullanılmaz |
| B6 | TalkBack ile kısa kontrol | ⬜ | Testlerde etiketler doğrulandı; gerçek cihazda elle deneme |

## C. Sonra

- iOS (Mac/Xcode gerekir), kalıcı hesap, admin paneli.
