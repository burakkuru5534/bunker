# Yayın kontrol listesi (REL-008)

Google Play (Android) ilk yayın için. Durum: ✅ hazır · 🟡 kısmen · ⬜ yapılmadı · ❓ karar bekliyor. Güncelleme: 2026-11-07.

## A. Yayını engelleyenler

| # | Madde | Durum | Not |
|---|---|---|---|
| A1 | Sunucu barındırma kararı | ✅ | Cloud Run, en düşük maliyet (ADR-0011; cost-model.md §0). Render Free uygulanmadı (ADR-0013) |
| A2 | Sunucu dağıtımı | 🟡 | `backend/Dockerfile` ✅, test servisi `bunker-api-test` ✅ (europe-west1; DB bağlı, duman testi ve 4 botla oyun geçti; `config/test.json` gerçek adreste). Kalan: prod servisi `bunker-api` (`APP_ENV=prod`, prod `DATABASE_URL`, `MIN_APP_VERSION`), duman testi (`/health`, `/health/ready`) |
| A3 | Prod veritabanı | ⬜ | `bunker_prod` şeması: `dbsetup` ile `bunker_prod_app` / `bunker_prod_migrator` rolleri, `migrate` ile 0001–0008 |
| A4 | Mobil prod adresleri | ⬜ | `apps/mobile/config/prod.json` şu an `example.invalid`; dağıtım adresi (https/wss) yazılır |
| A5 | Uygulama adı | ❓ | Logo "BUNKER — Son Kalanlar"; uygulama etiketi, `appTitle`, yasal sayfalar "Sığınak". Tek ad seçilip hepsi eşitlenir (Play adı ≤ 30 karakter) |
| A6 | Release imzası | ⬜ | Release şu an **debug anahtarıyla** imzalanıyor (`android/app/build.gradle.kts`). Yükleme anahtarı (keystore) + `key.properties` (git dışı) + `signingConfigs.release`; Play App Signing açılır. Keystore ve şifreleri yedeklenir — kaybolursa güncelleme yüklenemez |
| A7 | Sürüm ve paket | ⬜ | `pubspec.yaml` 0.1.0+1 → 1.0.0+1; `flutter build appbundle --dart-define-from-file=config/prod.json`; hedef API 36 ✅ (Flutter varsayılanı; Play şartı 31 Ağustos 2026'dan beri) |
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
