# Yol Haritası ve Görevler

> Çalışma prensibi: İncele → Planla → Tek küçük görevi uygula → Test et → Raporla → Kullanıcı onayı → Sonraki görev.
> Her görev ayrı onay gerektirir. Bir görev yalnızca kabul kriterleri sağlandığında **Done** olur.

**Durumlar:** Backlog · Ready · In Progress · Blocked · Awaiting User Approval · Done
**Öncelik:** P0 = MVP için zorunlu · P1 = MVP için önemli · P2 = MVP sonrası / ertelendi

Ayrıntılı kapsam, kabul kriterleri ve testler, görev **Ready** durumuna geçtiğinde ilgili bölüme eklenir.

---

## Faz 0 — Keşif ve ürün tanımı

| ID | Başlık | Öncelik | Bağımlılık | Durum |
|---|---|---|---|---|
| INIT-001 | Mevcut projeyi incele | P0 | — | Done |
| INIT-002 | Teknik ortamı doğrula | P0 | INIT-001 | Done |
| INIT-003 | Bağımlılıkları ve konfigürasyonları incele | P0 | INIT-001 | Done |
| GAME-001 | MVP kapsamını kesinleştir | P0 | INIT-001 | Done |
| GAME-002 | Oyuncu sayısı ve sığınak kapasitesi | P0 | GAME-001 | Done |
| GAME-003 | Oyun aşamaları ve kazanma koşulları | P0 | GAME-002 | Done |
| GAME-004 | Kart kategorileri ve gizlilik kuralları | P0 | GAME-003 | Done |
| GAME-005 | İlk felaket senaryoları | P0 | GAME-004 | Done |
| GAME-006 | Süreler, oylama ve beraberlik kuralları | P0 | GAME-003 | Done |
| DOC-001 | Doküman iskeleti ve kararların kaydı | P0 | GAME-006 | Done |
| ARCH-001 | Sistem mimarisi | P0 | DOC-001 | Done |
| ARCH-002 | WebSocket ve PostgreSQL veri akışı | P0 | ARCH-001 | Done |
| ARCH-003 | Güvenlik ve maliyet kararları | P0 | ARCH-001 | Done |
| ARCH-004 | API sözleşmeleri | P0 | ARCH-002 | Done |
| ARCH-005 | Cloud Run ve Supabase bağlantı modeli | P0 | ARCH-001 | Done |

### Tamamlanma notları

- **INIT-001:** Repo boş (yalnızca `.git`, commit yok). Branch `main`, remote `origin → github.com/burakkuru5534/bunker` (SSH). Supabase/GCP yapılandırması yok.
- **INIT-002:** Flutter 3.47.1, Dart 3.13.1, Go 1.27.0, Android SDK 36.0.0 doğrulandı. Android lisans adımı yeni Android CLI ile gereksiz; gerçek derleme MOB-001'de doğrulanacak. Testler bulut çalışma alanında aynı sürümlerle (Go 1.27.0, Flutter 3.47.1, yerel PostgreSQL) çalıştırılacak — kullanıcı onaylı.
- **INIT-003:** İncelenecek bağımlılık/konfigürasyon yok.
- **GAME-001…006:** Kararlar [product-requirements.md](product-requirements.md) ve [game-design.md](game-design.md) içinde.
- **DOC-001:** README, .gitignore, ürün gereksinimleri, oyun tasarımı ve yol haritası oluşturuldu.
- **ARCH-001:** [architecture.md](architecture.md) ve ADR-0001…0006 yazıldı. Kararlar: modül yapısı (4 sapma onaylı), satır kilidi + sürüm + action_id, ayrı test/prod ortamları, yerelde Docker Desktop ile PostgreSQL.
- **ARCH-002:** Komutlar REST, WS yalnızca sunucu → istemci ([ADR-0007](decisions/0007-rest-commands-ws-push.md)). Veri akışı [architecture.md §10](architecture.md), tablo taslağı [data-model.md](data-model.md).
- **ARCH-003:** [security.md](security.md), [cost-model.md](cost-model.md), [ADR-0008](decisions/0008-opaque-session-tokens.md). Maliyet kararları: Supabase Free + tek proje (test/prod şema izolasyonu, [ADR-0009](decisions/0009-single-supabase-schema-isolation.md)); Cloud Run `europe-west1`, Supabase `eu-central-1`.
- **ARCH-004:** [api.md](api.md) — REST uç noktaları, hata kodları, doğrulama kuralları, WS bağlantı/kapanış kodları, olay payload'ları. `/votes` → `/actions` içine birleştirildi; `PATCH/DELETE /v1/me`, `kick`, `rematch`, `settings`, `messages`, `reports`, `blocks` eklendi.
- **ARCH-005:** [architecture.md §11](architecture.md), [ADR-0010](decisions/0010-supavisor-session-mode.md) (Supavisor session mode, IPv4, pgxpool prod 5 / test 3), [ADR-0011](decisions/0011-cloud-run-configuration.md) (instance-based, min 0 / max 1, 1 vCPU / 512 MiB, timeout 3600 sn, europe-west1).

### DOC-001 — Doküman iskeleti

- **Kapsam:** `README.md`, `.gitignore`, `docs/product-requirements.md`, `docs/game-design.md`, `docs/roadmap.md`.
- **Kabul kriterleri:** GAME-001…006 kararlarının tümü dokümanlarda yer alır; secret içermez; `.gitignore` `.env` ve anahtar dosyalarını dışlar.
- **Testler:** Manuel inceleme (kod yok).

### ARCH-001 — Sistem mimarisi

- **Kapsam:** Bileşenler, backend modül sınırları, Flutter feature yapısı, dağıtım modeli; `docs/architecture.md` ve ilk ADR'ler (`docs/decisions/`).
- **Kabul kriterleri:** Kullanıcı mimariyi onaylar; alternatifler ve sınırlamalar yazılıdır.

---

## Faz 1 — UI/UX

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| UI-001 | Marka ve görsel dil ([ui/brand.md](ui/brand.md)) | P1 | Done |
| UI-002 | Ekran haritası ([ui/screen-map.md](ui/screen-map.md)) | P0 | Done |
| UI-003 | Tasarım sistemi ([ui/design-system.md](ui/design-system.md)) | P1 | Done |
| UI-004 | Ana ekran | P0 | Done |
| UI-005 | Oda oluşturma | P0 | Done |
| UI-006 | Odaya katılma | P0 | Done |
| UI-007 | Lobi | P0 | Done |
| UI-008 | Karakter kartları | P0 | Done |
| UI-009 | Tartışma ekranı | P0 | Done |
| UI-010 | Oylama ekranı | P0 | Done |
| UI-011 | Sonuç ekranı | P0 | Done |
| UI-012 | Hata ve bağlantı durumları | P0 | Done |

> UI-004…012 tek bir görsel mockup setinde hazırlandı (13 telefon ekranı, Claude Design kanvası "Sığınak Ekran Mockupları"). Kaynak: [ui/screen-map.md](ui/screen-map.md), [ui/design-system.md](ui/design-system.md).

## Faz 2 — Flutter temeli

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| MOB-001 | Flutter yapılandırması (Android derlemesi doğrulama dahil) | P0 | Awaiting User Approval |
| MOB-002 | Ortam konfigürasyonları | P0 | Awaiting User Approval |
| MOB-003 | Feature-first mimari ([apps/mobile/ARCHITECTURE.md](../apps/mobile/ARCHITECTURE.md)) | P0 | Awaiting User Approval |
| MOB-004 | Riverpod | P0 | Backlog |
| MOB-005 | go_router | P0 | Backlog |
| MOB-006 | Dio | P0 | Backlog |
| MOB-007 | WebSocket istemcisi | P0 | Backlog |
| MOB-008 | Tasarım sistemi bileşenleri | P1 | Backlog |
| MOB-009 | Hata yönetimi | P0 | Backlog |
| MOB-010 | Güvenli oturum depolama | P0 | Backlog |
| MOB-011 | Widget testleri | P0 | Backlog |
| MOB-012 | CI doğrulaması | P1 | Backlog |

- **MOB-001:** `apps/mobile` oluşturuldu (Flutter 3.47.1). Dart paketi `siginak`, **applicationId `com.bunker.app`** (kullanıcı kararı, kalıcı), uygulama adı "Sığınak", yalnızca dikey, sürüm 0.1.0+1. `dart format`, `flutter analyze`, `flutter test` bulut çalışma alanında geçti. Android derlemesi kullanıcı cihazında doğrulanacak.
- **MOB-002:** `--dart-define-from-file` ile `local`/`test`/`prod` yapılandırması (`apps/mobile/config/`), `AppConfig` doğrulaması (test/prod'da https/wss zorunlu), düz http yalnızca debug derlemede, release manifest'e `INTERNET` izni. `local.json` git dışı. 10 test geçti. `test.json`/`prod.json` adresleri Cloud Run kurulunca güncellenecek.
- **MOB-003:** `lib/app`, `lib/core`, `lib/features` yapısı; `SiginakApp` → `app/app.dart`, geçici ekran → `features/home/presentation/home_screen.dart`. Bağımlılık kuralları `test/architecture/import_rules_test.dart` ile otomatik denetleniyor. `analysis_options.yaml` sıkılaştırıldı (strict modlar; package import, print, unawaited futures = hata). 14 test geçti.

## Faz 3 — Go backend ve PostgreSQL

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| BE-001 | Go servis iskeleti | P0 | Backlog |
| BE-002 | HTTP router ve middleware | P0 | Backlog |
| BE-003 | Yapılandırılmış loglama | P0 | Backlog |
| BE-004 | Standart API hata formatı | P0 | Backlog |
| BE-005 | Health endpoint | P0 | Backlog |
| DB-001 | Supabase PostgreSQL bağlantısı | P0 | Backlog |
| DB-002 | Migration altyapısı | P0 | Backlog |
| DB-003 | İlk şema | P0 | Backlog |
| DB-004 | Sorgu ve indeks tasarımı | P0 | Backlog |
| DB-005 | Bağlantı havuzu ve limitleri | P0 | Backlog |
| AUTH-001 | Misafir oturumu tasarımı | P0 | Backlog |
| AUTH-002 | Oturum oluşturma ve doğrulama | P0 | Backlog |
| AUTH-003 | REST ve WebSocket yetkilendirmesi | P0 | Backlog |

## Faz 4 — Gerçek zamanlı altyapı

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| RT-001 | WebSocket sunucusu | P0 | Backlog |
| RT-002 | Bağlantı kimlik doğrulaması | P0 | Backlog |
| RT-003 | Oda üyeliği kontrolü | P0 | Backlog |
| RT-004 | Oda bazlı bağlantı yönetimi | P0 | Backlog |
| RT-005 | Mesaj şemaları ve doğrulama | P0 | Backlog |
| RT-006 | Oda içi olay yayını | P0 | Backlog |
| RT-007 | Bağlantı kopması ve yeniden bağlanma | P0 | Backlog |
| RT-008 | Bağlantı temizliği ve kaynak yönetimi | P0 | Backlog |
| RT-009 | Rate limiting | P0 | Backlog |
| RT-010 | Eşzamanlılık ve bağlantı testleri | P0 | Backlog |
| RT-011 | Cloud Run WebSocket kısıtlarının doğrulanması | P0 | Backlog |
| RT-012 | Gerçek zamanlı iletişim maliyet değerlendirmesi | P1 | Backlog |

## Faz 5 — Oda ve lobi

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| ROOM-001 | Oda modeli | P0 | Backlog |
| ROOM-002 | Davet kodu üretimi | P0 | Backlog |
| ROOM-003 | Oda oluşturma API'si | P0 | Backlog |
| ROOM-004 | Odaya katılma API'si | P0 | Backlog |
| ROOM-005 | Odadan ayrılma | P0 | Backlog |
| ROOM-006 | Oda üyeliği ve yetkilendirme | P0 | Backlog |
| ROOM-007 | Hazır ol mekanizması | P0 | Backlog |
| ROOM-008 | Oda ayarlarının senkronizasyonu (kapasite 1/2) | P0 | Backlog |
| ROOM-009 | Oda sahibi ayrıldığında yetki devri | P0 | Backlog |
| ROOM-010 | Boş oda temizliği | P1 | Backlog |
| ROOM-011 | Flutter oda akışları | P0 | Backlog |
| ROOM-012 | Çoklu oyuncu testleri | P0 | Backlog |

## Faz 6 — Kart ve senaryo sistemi

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| CARD-001 | Kart veri modeli (denge etiketi dahil) | P0 | Backlog |
| CARD-002 | Meslek kartları | P0 | Backlog |
| CARD-003 | Sağlık durumu kartları | P0 | Backlog |
| CARD-004 | Hobi ve beceri kartları | P0 | Backlog |
| CARD-005 | Eşya kartları | P0 | Backlog |
| CARD-006 | Gizli sır kartları | P0 | Backlog |
| CARD-007 | Felaket senaryosu modeli ve 5 senaryo metni | P0 | Backlog |
| CARD-008 | Rastgele kart dağıtımı | P0 | Backlog |
| CARD-009 | Oyuncuya özel gizli veri | P0 | Backlog |
| CARD-010 | Kart açıklama kuralları | P0 | Backlog |
| CARD-011 | İçerik doğrulama | P0 | Backlog |
| CARD-012 | Kart denge testleri | P1 | Backlog |

## Faz 7 — Oyun motoru

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| ENG-001 | Oyun durum makinesi | P0 | Backlog |
| ENG-002 | Durum geçişi doğrulama | P0 | Backlog |
| ENG-003 | Maç başlatma | P0 | Backlog |
| ENG-004 | Kart dağıtımı | P0 | Backlog |
| ENG-005 | Kart açıklama sırası | P0 | Backlog |
| ENG-006 | Tartışma aşaması | P0 | Backlog |
| ENG-007 | Tur zamanlayıcıları (kalıcı deadline) | P0 | Backlog |
| ENG-008 | Oylama | P0 | Backlog |
| ENG-009 | Oylama sonucu ve eşitlik bozma | P0 | Backlog |
| ENG-010 | Eleme | P0 | Backlog |
| ENG-011 | Kazanma koşulları | P0 | Backlog |
| ENG-012 | Maç sonu | P0 | Backlog |
| ENG-013 | Yeniden bağlanma ve oyun kurtarma | P0 | Backlog |
| ENG-014 | İdempotent eylemler | P0 | Backlog |
| ENG-015 | Gizli bilgi izolasyonu | P0 | Backlog |
| ENG-016 | Oyun motoru birim testleri | P0 | Backlog |

## Faz 8 — Oyun UI entegrasyonu

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| PLAY-001 | Lobi ve başlatma | P0 | Backlog |
| PLAY-002 | Kişisel karakter kartları | P0 | Backlog |
| PLAY-003 | Açıklanmış kartlar | P0 | Backlog |
| PLAY-004 | Tartışma ekranı | P0 | Backlog |
| PLAY-005 | Mesaj gönderimi | P0 | Backlog |
| PLAY-006 | Tur sayacı | P0 | Backlog |
| PLAY-007 | Oylama arayüzü | P0 | Backlog |
| PLAY-008 | Eleme sonucu | P0 | Backlog |
| PLAY-009 | Yeniden bağlantı deneyimi | P0 | Backlog |
| PLAY-010 | Maç sonu ekranı | P0 | Backlog |
| PLAY-011 | Hata ve yüklenme durumları | P0 | Backlog |
| PLAY-012 | Küçük ekran ve erişilebilirlik testleri | P1 | Backlog |

## Faz 9 — Güvenlik ve moderasyon

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| SAFE-001 | İstek ve girdi doğrulama | P0 | Backlog |
| SAFE-002 | Rate limiting | P0 | Backlog |
| SAFE-003 | Oda izolasyonu | P0 | Backlog |
| SAFE-004 | Gizli kart erişim testleri | P0 | Backlog |
| SAFE-005 | Oyuncu raporlama | P0 | Backlog |
| SAFE-006 | Oyuncu engelleme | P0 | Backlog |
| SAFE-007 | Oda sahibi moderasyonu (lobiden atma) | P0 | Backlog |
| SAFE-008 | Gizlilik politikası ve kullanım koşulları | P0 | Backlog |
| SAFE-009 | Veri saklama ve silme (sohbet 30 gün) | P0 | Backlog |
| SAFE-010 | Secret yönetimi | P0 | Backlog |
| SAFE-011 | Güvenlik logları | P1 | Backlog |
| SAFE-012 | Mağaza güvenlik ve yaş derecelendirmesi | P0 | Backlog |

## Faz 10 — Test, beta ve yayın

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| QA-001 | Oyun motoru testleri | P0 | Backlog |
| QA-002 | API entegrasyon testleri | P0 | Backlog |
| QA-003 | Çoklu cihaz testleri | P0 | Backlog |
| QA-004 | Ağ kesintisi testleri | P0 | Backlog |
| QA-005 | Arka plan ve yeniden bağlanma testleri | P0 | Backlog |
| QA-006 | Backend yeniden başlatma testleri | P0 | Backlog |
| QA-007 | Yük ve maliyet ölçümleri | P1 | Backlog |
| QA-008 | Güvenlik testleri | P0 | Backlog |
| REL-001 | Uygulama ikonları ve mağaza görselleri | P0 | Backlog |
| REL-002 | Gizlilik politikası ve destek sayfası | P0 | Backlog |
| REL-003 | Production konfigürasyonu | P0 | Backlog |
| REL-004 | Crash ve performans takibi | P1 | Backlog |
| REL-005 | Google Play test dağıtımı (TestFlight ertelendi) | P0 | Backlog |
| REL-006 | Kapalı beta | P0 | Backlog |
| REL-007 | Beta sorunlarının giderilmesi | P0 | Backlog |
| REL-008 | Yayın kontrol listesi | P0 | Backlog |

## Faz 11 — MVP sonrası (yalnızca beta değerlendirmesinden sonra)

| ID | Başlık | Öncelik | Durum |
|---|---|---|---|
| POST-001 | iOS derlemesi ve TestFlight (bulut CI ile) | P2 | Backlog |
| POST-002 | Herkese açık oda listesi | P2 | Backlog |
| POST-003 | Arkadaş listesi | P2 | Backlog |
| POST-004 | Oyuncu profili ve istatistikleri | P2 | Backlog |
| POST-005 | Yeni senaryo paketleri ve kart kategorileri | P2 | Backlog |
| POST-006 | Sesli sohbet | P2 | Backlog |
| POST-007 | Paylaşılabilir maç sonuçları | P2 | Backlog |
| POST-008 | Kozmetik / premium özellikler | P2 | Backlog |
| POST-009 | Yayıncı modu, gelişmiş eşleştirme | P2 | Backlog |
