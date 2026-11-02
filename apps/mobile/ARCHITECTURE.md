# Mobil Mimari — feature-first

> Kaynak: MOB-003. Genel mimari: [../../docs/architecture.md](../../docs/architecture.md) §7. Ekranlar: [../../docs/ui/screen-map.md](../../docs/ui/screen-map.md).

## 1. Klasör yapısı

```
lib/
  main.dart                    Yalnızca yapılandırmayı okur ve SiginakApp'i başlatır
  app/                         Uygulamayı birleştiren katman
    app.dart                   MaterialApp, tema, router bağlama
    router.dart                go_router rotaları (MOB-005)
    theme/                     app_colors, app_spacing, app_typography, app_theme, game_theme (MOB-008)
  core/                        Feature'lardan bağımsız altyapı
    config/                    AppConfig (MOB-002)
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
- **Sabit renk / boşluk değeri** widget içine yazılmaz; tema token'ları kullanılır (MOB-008).
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

## 6. Adlandırma

| Öğe | Kural | Örnek |
|---|---|---|
| Dosya | `snake_case.dart` | `lobby_player_tile.dart` |
| Ekran | `<Ad>Screen` | `LobbyScreen` |
| Widget | Açıklayıcı isim, tasarım sistemindeki adla aynı | `CountdownTimer` |
| Provider | `<ad>Provider` | `roomStateProvider` |
| Repository | `<Ad>Repository` | `RoomRepository` |
| DTO | `<Ad>Dto` | `RoomViewDto` |

## 7. Testler

- `test/` klasörü `lib/` yapısını aynen yansıtır: `lib/features/home/presentation/home_screen.dart` → `test/features/home/presentation/home_screen_test.dart`.
- `test/architecture/` bağımlılık kurallarını denetler.
- `test/helpers/` test yardımcılarıdır (ör. `pumpWithProviders`).
- Her görevde: `dart format --set-exit-if-changed .`, `flutter analyze`, `flutter test`.

## 8. Statik analiz

`analysis_options.yaml`: `strict-casts`, `strict-inference`, `strict-raw-types` açık. `always_use_package_imports`, `avoid_print`, `unawaited_futures` **hata** seviyesindedir. Loglama için `print` yerine `debugPrint` / ileride log yardımcısı kullanılır.
