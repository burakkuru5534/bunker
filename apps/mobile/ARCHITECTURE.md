# Mobil Mimari — feature-first

> Kaynak: MOB-003. Genel mimari: [../../docs/architecture.md](../../docs/architecture.md) §7. Ekranlar: [../../docs/ui/screen-map.md](../../docs/ui/screen-map.md).

## 1. Klasör yapısı

```
lib/
  main.dart                    Yalnızca yapılandırmayı okur ve SiginakApp'i başlatır
  app/                         Uygulamayı birleştiren katman
    app.dart                   MaterialApp.router, tema bağlama
    router/                    go_router rotaları ve yönlendirme (MOB-005)
  core/                        Feature'lardan bağımsız altyapı
    config/                    AppConfig, kAppVersion (MOB-002, MOB-006)
    theme/                     app_colors, app_spacing, app_typography, app_theme, game_theme, font_licenses (MOB-008)
    realtime/                  WebSocket istemcisi (MOB-007)
    network/                   Dio istemcisi, WS istemcisi, hata eşleme (MOB-006/007/009)
    storage/                   Güvenli oturum depolama (MOB-010)
    error/                     Uygulama hata modeli, hata kodu → l10n metni (MOB-009)
    ui/                        Ortak bileşenler: AppPrimaryButton, PlayerAvatar, ConnectionBanner… (MOB-008)
    l10n/                      Türkçe metinler (ARB)
  features/
    session/                   Misafir oturumu, takma ad, /welcome
    home/                      Ana ekran, koda katıl
    room/                      Oda kabuğu, lobi
    game/                      Senaryo, kart açma, tartışma, oylama, eleme, sonuç
    chat/                      Sohbet paneli
    moderation/                Raporla, engelle, engellenenler
    settings/                  Ayarlar, hesap silme
```

Her feature içinde:

```
features/<ad>/
  <ad>.dart                    (isteğe bağlı) Diğer feature'lara açık tek giriş noktası
  data/                        API çağrıları, DTO'lar (JSON ↔ model), repository
  presentation/                Ekranlar, widget'lar, Riverpod provider'ları
    <ad>_screen.dart
    widgets/
  domain/                      Yalnızca gerçekten gerekirse (saf modeller / kurallar)
```

Klasörler **ihtiyaç doğdukça** oluşturulur; boş klasör veya yer tutucu dosya eklenmez.

## 2. Bağımlılık kuralları

```
main.dart → app/ → features/ → core/
                 ↘────────────↗
```

| Kural | Neden | Denetim |
|---|---|---|
| `core/` → `app/` veya `features/` içe aktaramaz | Altyapı tek yönlü bağımlı kalmalı | `test/architecture/import_rules_test.dart` |
| `features/` → `app/` içe aktaramaz | Döngüsel bağımlılık oluşmasın | aynı test |
| Bir feature başka bir feature'ın **içini** içe aktaramaz; yalnızca `features/<ad>/<ad>.dart` giriş noktasını kullanabilir | Feature'lar bağımsız değişebilsin | aynı test |
| Tüm içe aktarmalar `package:siginak/...` biçiminde | Tutarlılık, taşıma kolaylığı | `always_use_package_imports` (hata) |

Ortak ihtiyaç iki feature'da birden doğarsa kod `core/`'a taşınır.

## 3. Sorumluluklar

- **İstemci oyun kuralı hesaplamaz.** Ekranlar yalnızca sunucudan gelen durumu (snapshot + WS olayları) gösterir; geçerlilik kontrolü kullanıcı deneyimi içindir, karar sunucudadır.
- **Widget'lar** veri çekmez; provider'lardan okur.
- **Provider'lar** (MOB-004) repository'leri çağırır ve UI durumunu tutar.
- **Repository'ler** (`data/`) REST/WS çağrılarını yapar, DTO'ları modele çevirir, hata kodlarını uygulama hatasına eşler.
- **Sabit renk / boşluk değeri** widget içine yazılmaz; tema token'ları kullanılır (§8).
- **Metinler** l10n dosyalarından gelir (ekranlar oluşturulurken eklenecek).

## 4. Durum yönetimi (Riverpod)

Paket: `flutter_riverpod` **3.4.3** (Dart `^3.12.0`). Kod üretici (riverpod_generator / build_runner) **kullanılmaz**: ek bağımlılık ve derleme adımı getirir; elle yazılan provider'lar bu ölçekte yeterince kısadır.

| Kural | Ayrıntı |
|---|---|
| Kök | `main.dart` uygulamayı `ProviderScope` ile sarar; ortam yapılandırması `appConfigProvider.overrideWithValue(...)` ile verilir |
| Provider türleri | Bağımlılık / servis → `Provider`; değişen ekran durumu → `Notifier` / `NotifierProvider`; sunucudan yüklenen durum → `AsyncNotifier` / `FutureProvider`; WS akışı → `StreamProvider` |
| Eski API | `StateProvider`, `StateNotifierProvider`, `ChangeNotifierProvider` (`legacy.dart`) **kullanılmaz** |
| Yaşam süresi | Ekrana özgü durum `.autoDispose`; uygulama boyu yaşayan servisler (oturum, API istemcisi, WS bağlantısı) kalıcı |
| Okuma | `build` içinde `ref.watch`; olay işleyicilerinde (`onPressed` vb.) `ref.read` |
| Konum | Servis/repository provider'ı → ilgili `data/`; ekran durumu → ilgili `presentation/`; feature'lar arası ortak → `core/` |
| Widget | Durum okuyan widget'lar `ConsumerWidget` / `ConsumerStatefulWidget` |
| Test | `ProviderScope(overrides: ...)` veya `ProviderContainer`; ortak yardımcı `test/helpers/pump_app.dart` |
| Zorunlu override | Yalnızca çalışma zamanında belli olan değerler (`appConfigProvider`) override edilmeden okunursa hata verir |

## 5. Yönlendirme (go_router)

Paket: `go_router` `^18.1.0` (Dart `^3.12.0`, Flutter `>=3.44.0`). Rotalar [docs/ui/screen-map.md](../../docs/ui/screen-map.md) §2 ile birebir aynıdır.

| Dosya | Görev |
|---|---|
| `app/router/app_routes.dart` | Rota yolları ve `room(id)` yardımcısı, oda kimliği (UUID) doğrulaması |
| `app/router/app_redirect.dart` | **Saf** yönlendirme fonksiyonu: güncelleme → oturum → aktif oda kuralları (birim testli) |
| `app/router/app_router.dart` | Tek `GoRouter` (`appRouterProvider`); oturum veya güncelleme durumu değişince `refreshListenable` ile yönlendirme yeniden değerlendirilir |

Kurallar:
- **Ekranlar rota yollarını bilmez.** Başka bir ekrana geçiş, ekrana geri çağrı (`onJoin`, `onOpenSettings` …) olarak verilir ve `app_router.dart` içinde bağlanır. Böylece feature'lar `app/`'e bağımlı olmaz.
- Oturum / güncelleme gibi **durum kaynaklı geçişler** doğrudan `context.go` ile değil, ilgili provider'ın durumu değiştirilerek yapılır; router `redirect` ile doğru ekrana götürür.
- Oda ekranı tek rotadır (`/room/:roomId`); lobi/oyun/sonuç görünümleri sunucu durumuna göre ekranın içinde seçilir.
- Geçersiz oda kimliği ve bilinmeyen adresler `NotFoundScreen` gösterir.
- Feature'ın dışarıya açtıkları `features/<ad>/<ad>.dart` dosyasından içe aktarılır (ör. `features/session/session.dart`).

## 6. Ağ (REST — Dio)

Paket: `dio` `^5.11.1`. Sözleşme: [docs/api.md](../../docs/api.md).

| Dosya | Görev |
|---|---|
| `core/network/api_client.dart` | `ApiClient`: `get/post/patch/delete`; taban adres, `X-App-Version`, varsa `Authorization: Bearer`; zaman aşımları (bağlantı 10 sn, gönderme 10 sn, alma 15 sn) |
| `core/network/api_exception.dart` | `ApiException` (`kind`, `statusCode`, `code`, `requestId`, `retryAfter`, `fieldErrors`) |
| `core/network/api_error_mapper.dart` | `DioException` → `ApiException` (zaman aşımı, ağ, 401, 426, 429, standart hata gövdesi) |
| `core/network/api_client_provider.dart` | Uygulama boyu tek istemci (`apiClientProvider`) |
| `core/network/auth_token_provider.dart` | Token okuyucu ve 401 işleyicisi — MOB-010 / AUTH-002 override eder |
| `core/config/app_version.dart` | `kAppVersion` — `pubspec.yaml` ile eşitliği testle denetlenir |

Kurallar:
- Repository'ler yalnızca `ApiClient` kullanır; `Dio`'yu doğrudan kullanmaz.
- Tüm hatalar `ApiException`'dır; UI metni `code`/`kind`'a göre l10n'dan seçilir (MOB-009). `debugMessage` kullanıcıya gösterilmez.
- **426** → `updateRequiredProvider` işaretlenir → router `/update`'e kilitler. **401** → `unauthorizedHandlerProvider` çağrılır.
- Otomatik yeniden deneme **yok**: oyun eylemleri `action_id` ile idempotent olsa da tekrar kararı çağıran katmandadır.
- Loglama yalnızca debug derlemede ve yalnızca `yöntem yol → durum (süre)`; **başlık, token ve gövde loglanmaz.**
- Testlerde ağa çıkılmaz: `test/helpers/fake_http_adapter.dart`.

## 7. Gerçek zamanlı (WebSocket)

Ek paket yok: `dart:io` `WebSocket` (mobil hedef; `Authorization` başlığı ve ping/pong destekli). Sözleşme: [docs/api.md](../../docs/api.md) §5, [ADR-0006](../../docs/decisions/0006-snapshot-on-reconnect.md), [ADR-0007](../../docs/decisions/0007-rest-commands-ws-push.md).

| Dosya | Görev |
|---|---|
| `core/realtime/realtime_client.dart` | `RealtimeClient`: tek odaya bağlantı, olay akışı (`events`), durum akışı (`states`), `state_version` takibi, yeniden bağlanma |
| `core/realtime/ws_envelope.dart` | v1 zarfı çözümleme; geçersiz/başka sürüm mesajlar yok sayılır |
| `core/realtime/ws_socket.dart` | Test edilebilir soket soyutlaması ve `dart:io` uygulaması (bağlantı zaman aşımı 10 sn, istemci ping'i 30 sn) |
| `core/realtime/realtime_client_provider.dart` | `realtimeClientProvider`, `realtimeStateProvider` |

Davranış:
- Adres `wss://…/v1/ws?room_id=<id>`; token **yalnızca** `Authorization` başlığında.
- WS yalnızca sunucu → istemci: istemci sadece `{"v":1,"type":"sync"}` gönderir; komutlar REST ile gider.
- `state_version`: bağlantı/yeniden bağlantı sonrası `session.snapshot` beklenir; sonra yalnızca `bilinen + 1` kabul edilir, kopya/eski atılır, **boşlukta `sync`** istenir ve yeni snapshot'a kadar sürümlü olaylar atlanır. Sürümsüz olaylar (`chat.message`) her zaman iletilir.
- Yeniden bağlanma: 1001/1006/1012/4009 ve bağlantı hatalarında üstel geri çekilme (0,5 → 1 → 2 → … en fazla 10 sn, ±%20); başarılı bağlantıda sayaç sıfırlanır.
- Kalıcı durma: **4001** (oturum → `unauthorizedHandlerProvider`), **4003** (üye değil/atıldı), **4008** (yerini yeni bağlantı aldı), **4026** (sürüm → `/update`).
- `onAppResumed()`: bağlıysa `sync`, değilse beklemeden yeniden bağlan (oda ekranı yaşam döngüsüne bağlayacak — PLAY-009).
- Mesaj içeriği, başlık ve token loglanmaz.
- Testlerde gerçek soket yok: `test/helpers/fake_ws.dart` + `fake_async`.

## 8. Tema ve tasarım token'ları

Kaynak: [docs/ui/brand.md](../../docs/ui/brand.md), [docs/ui/design-system.md](../../docs/ui/design-system.md).

> Tema `app/` yerine **`core/theme/`** altındadır: hem `core/ui` bileşenleri hem feature'lar kullandığından `app/`'te olsaydı bağımlılık kuralları (§2) bozulurdu.

| Dosya | İçerik |
|---|---|
| `app_colors.dart` | Renk token'ları (yüzey, metin, vurgu, kategori, avatar döngüsü) |
| `app_spacing.dart` | `AppSpacing` (4'ün katları, `minTouchTarget` 48), `AppRadius`, `AppMotion` |
| `app_typography.dart` | `AppFonts` (Oswald / Inter), `buildTextTheme`, `turkishUpperCase` / `turkishLowerCase` |
| `game_theme.dart` | `ThemeExtension<GameTheme>`: kategori stili (renk + ikon + etiket), avatar rengi, danger/success/hope; `context.gameTheme` |
| `app_theme.dart` | `buildAppTheme()`: koyu M3 `ColorScheme` ve bileşen temaları (düğme 52 dp, giriş alanı, chip, snackbar, alt sayfa, diyalog…) |
| `font_licenses.dart` | OFL lisanslarını `LicenseRegistry`'ye ekler |

Kurallar:
- Widget içinde **sabit renk / boyut yazılmaz**: `Theme.of(context)`, `context.gameTheme`, `AppSpacing`, `AppRadius` kullanılır. `AppColors` yalnızca tema kurulumunda ve testlerde doğrudan okunur.
- Büyük harfli metinlerde `toUpperCase()` değil `turkishUpperCase()`.
- Yazı tipleri `assets/fonts/` altında gömülüdür (alt kümeli, ≈430 KB; ayrıntı `assets/fonts/README.md`).
- Kontrast oranları `test/core/theme/contrast_test.dart` ile denetlenir (≥ 4,5:1).

## 9. Adlandırma

| Öğe | Kural | Örnek |
|---|---|---|
| Dosya | `snake_case.dart` | `lobby_player_tile.dart` |
| Ekran | `<Ad>Screen` | `LobbyScreen` |
| Widget | Açıklayıcı isim, tasarım sistemindeki adla aynı | `CountdownTimer` |
| Provider | `<ad>Provider` | `roomStateProvider` |
| Repository | `<Ad>Repository` | `RoomRepository` |
| DTO | `<Ad>Dto` | `RoomViewDto` |

## 10. Testler

- `test/` klasörü `lib/` yapısını aynen yansıtır: `lib/features/home/presentation/home_screen.dart` → `test/features/home/presentation/home_screen_test.dart`.
- `test/architecture/` bağımlılık kurallarını denetler.
- `test/helpers/` test yardımcılarıdır (ör. `pumpWithProviders`).
- Her görevde: `dart format --set-exit-if-changed .`, `flutter analyze`, `flutter test`.

## 11. Statik analiz

`analysis_options.yaml`: `strict-casts`, `strict-inference`, `strict-raw-types` açık. `always_use_package_imports`, `avoid_print`, `unawaited_futures` **hata** seviyesindedir. Loglama için `print` yerine `debugPrint` / ileride log yardımcısı kullanılır.
