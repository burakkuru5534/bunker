# Tasarım Sistemi — Sığınak: Son Kalanlar

> Kaynak: UI-003 (2026-11-02). Renk, yazı tipi ve ses tonu: [brand.md](brand.md). Ekranlar: [screen-map.md](screen-map.md). Uygulama: MOB-008.

## 1. Token'lar

### Boşluk (4'ün katları)

| Token | dp | Kullanım |
|---|---|---|
| `space1` | 4 | İkon–metin arası |
| `space2` | 8 | Çip içi, liste satırı içi |
| `space3` | 12 | Kart iç boşluğu (küçük) |
| `space4` | 16 | **Ekran kenar boşluğu**, kart iç boşluğu |
| `space5` | 24 | Bölümler arası |
| `space6` | 32 | Büyük bölümler |
| `space8` | 48 | Ekran üst boşluğu (başlık ekranları) |

### Köşe yuvarlaklığı

| Token | dp | Kullanım |
|---|---|---|
| `radiusS` | 8 | Çip, rozet, giriş kutusu |
| `radiusM` | 12 | Düğme, sohbet balonu |
| `radiusL` | 16 | Kart, oyuncu kutusu, diyalog |
| `radiusXL` | 24 | Alt sayfa (bottom sheet) üst köşeleri |
| `radiusFull` | 999 | Avatar, hap biçimli çip |

### Yükselti

Koyu temada gölge yerine **yüzey tonu** kullanılır: `background` → `surface` → `surfaceHigh`. Gölge yalnızca diyalog ve alt sayfada (Material 3 varsayılanı).

### Kenarlık

- Varsayılan: 1 dp, `#2E333D` (`outline`).
- Seçili / vurgulu: 2 dp, `primary`.
- Oylama hedefi seçili: 2 dp, `danger`.

### Süre ve eğri (hareket)

| Token | Süre | Kullanım |
|---|---|---|
| `motionFast` | 150 ms | Basma geri bildirimi, renk geçişi |
| `motionMedium` | 300 ms | Kart çevirme, alt sayfa |
| `motionSlow` | 500 ms | Eleme solması, aşama geçişi |

Eğri: `Curves.easeOutCubic` (giriş), `Curves.easeInCubic` (çıkış). "Animasyonları azalt" açıksa süreler 0 (solma hariç).

## 2. Flutter karşılığı

```
apps/mobile/lib/core/theme/
  app_colors.dart        // brand.md renk sabitleri
  app_spacing.dart       // space*, radius*
  app_typography.dart    // Oswald / Inter TextTheme (Türkçe büyük harf yardımcıları)
  app_theme.dart         // ThemeData(useMaterial3: true, colorScheme: dark, ...)
  game_theme.dart        // ThemeExtension<GameTheme>: kategori renkleri/ikonları/etiketleri,
                         //   avatar renk döngüsü, danger/success/secondary vurguları
```

> Tema `core/theme/` altındadır (MOB-008a): hem `core/ui` hem feature'lar kullandığından `app/` altında olsaydı bağımlılık kuralları bozulurdu.

- `ColorScheme.dark(...)`: `primary`, `onPrimary`, `secondary`, `error = danger`, `surface`, `surfaceContainerHigh = surfaceHigh`, `onSurface = textPrimary`, `onSurfaceVariant = textSecondary`, `outline`.
- Oyuna özgü değerler `ThemeExtension<GameTheme>` içinde; widget'lar `Theme.of(context).extension<GameTheme>()!` ile okur. Sabit renk kodu widget içinde **yazılmaz**.
- Bileşen temaları `ThemeData` üzerinden: `filledButtonTheme`, `outlinedButtonTheme`, `inputDecorationTheme`, `chipTheme`, `snackBarTheme`, `bottomSheetTheme`, `dialogTheme`.

## 3. Bileşenler

Ortak bileşenler `lib/core/ui/` altında; yalnızca bir feature'ın kullandıkları o feature'ın `presentation/widgets/` klasöründe.

### Temel (core/ui)

| Bileşen | Açıklama | Durumlar |
|---|---|---|
| `AppPrimaryButton` | Turuncu dolu düğme, yükseklik 52 dp, tam genişlik seçeneği | normal, basılı, devre dışı (nedeni altta), yükleniyor (spinner, tekrar basılamaz) |
| `AppSecondaryButton` | Kenarlıklı düğme | aynı |
| `AppDangerButton` | Kırmızı kenarlıklı; ayrıl, hesap sil | aynı |
| `AppTextField` | Etiket, yardımcı metin, sayaç, hata metni | odak, hata, devre dışı |
| `InviteCodeInput` | 6 kutulu kod girişi; yapıştırma, otomatik büyük harf, alfabe filtresi | boş, kısmi, tam, hata |
| `PlayerAvatar` | Baş harf + döngü rengi; boyut S 28 / M 40 / L 64 | çevrimiçi, bağlantı koptu (gri + nokta), elendi (soluk + çizgi) |
| `StatusBadge` | Küçük hap: "Hazır", "Oda sahibi", "İzleyici", "Elendi", "Ayrıldı" | renkleri brand.md'den |
| `ConnectionBanner` | Üst şerit: yeniden bağlanıyor (turuncu) / bağlandı (yeşil, 2 sn) / bağlantı yok (kırmızı + Tekrar dene) | 3 durum |
| `ConfirmDialog` | Başlık, açıklama, birincil/ikincil eylem; tehlikeli varyant | normal, tehlikeli |
| `AppSnackBar` | Hata kodundan l10n metni | bilgi, hata |
| `LoadingView` / `ErrorView` | Tam ekran yükleniyor / hata + Tekrar dene | — |
| `EmptyState` | İkon + metin (ör. engellenen yok) | — |

### Oyun (features/*/presentation/widgets)

| Bileşen | Feature | Açıklama |
|---|---|---|
| `LobbyPlayerTile` | room | Avatar, takma ad, rozetler, hazır işareti; oda sahibinde "At" menüsü |
| `CapacitySelector` | room | 1 / 2 bölümlü seçici; sahibi değilse salt okunur |
| `InviteCodeCard` | room | Büyük Oswald kod + Kopyala (kopyalanınca "Kopyalandı" geri bildirimi) |
| `PhaseHeader` | game | Senaryo çipi, "Tur 2/8", aşama adı, `CountdownTimer` |
| `CountdownTimer` | game | `deadline − serverTime` ile hesap; son 10 sn kırmızı + haptic; erişilebilirlik etiketi "72 saniye kaldı" |
| `CategoryChip` | game | Kategori ikonu + renk + kısa metin ("Cerrah"); açılmamış kart için kilitli gri çip |
| `CharacterCard` | game | Tam kart: kategori başlığı, metin, açık/kapalı; `motionMedium` çevirme |
| `PlayerGameTile` | game | Avatar, ad, durum, açılmış `CategoryChip`'ler; seçilebilir (oylama) varyant |
| `MyCardsSheet` | game | Kendi 5 kartım (alt sayfa) |
| `RevealActionPanel` | game | Açılmamış kartlarımdan seçim; 1. turda yalnızca meslek etkin |
| `DiscussionPanel` | game | "Hazırım" + `ProgressCounter` |
| `VotePanel` | game | Hedef seçimi + onay; oy verildikten sonra kilitli durum |
| `TieBreakPanel` | game | İzleyici: aday seçimi; oyuncu: bekleme metni |
| `ProgressCounter` | game | "4/7 oy verildi" + ince ilerleme çubuğu |
| `EliminationReveal` | game | Elenen oyuncu, yöntem, oy sayıları, tüm kartları |
| `SpectatorBanner` | game | "İzleyicisin" şeridi |
| `ScenarioCard` | game | Senaryo başlığı, metin, süre, sığınak maddeleri, dış dünya |
| `ResultSummary` | game | Kazananlar / dışarıda kalanlar, tur özeti |
| `ChatPanel` | chat | Mesaj listesi + giriş; izleyicide salt okunur; 200 karakter sayacı |
| `ChatBubble` | chat | Gönderen avatarı + ad (kendi mesajım sağda), zaman |
| `ReportSheet` | moderation | Neden seçimi + not |

## 4. Durum kalıpları

Her veri gösteren ekran şu dört durumu tasarlar: **yükleniyor**, **içerik**, **boş**, **hata**. Gerçek zamanlı ekranlar ek olarak **bağlantı koptu** durumunu `ConnectionBanner` ile gösterir; içerik son bilinen durumla görünür kalır, eylem düğmeleri devre dışı olur.

## 5. Düzen kuralları

- Hedef ekran genişliği 360–430 dp (Android telefon); 320 dp'de taşma olmamalı (PLAY-012).
- Ekran kenar boşluğu `space4` (16 dp).
- Oyuncu ızgarası: 2 sütun; ≤4 oyuncuda 1 sütun büyük kutular.
- Alt eylem alanı her zaman görünür; klavye açılınca sohbet paneli yeniden boyutlanır.
- Yatay yön (landscape) MVP'de kilitli: yalnızca dikey.

## 6. Erişilebilirlik kontrol listesi (her bileşen için)

- [ ] Dokunma alanı ≥ 48×48 dp
- [ ] `Semantics` etiketi (ikon düğmeler, sayaç, kart çipleri)
- [ ] Renk tek başına anlam taşımıyor
- [ ] Metin ölçeği %200'de taşma yok
- [ ] "Animasyonları azalt"a uyuyor
