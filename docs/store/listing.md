# Google Play mağaza kaydı (REL-008 A10)

Play Console → **Büyüme → Mağaza varlığı → Ana mağaza girişi**. Dil: Türkçe (tr-TR). Metinler uygulamanın **bugünkü** özelliklerini anlatır (product-requirements.md); özellik eklenmedikçe vaat edilmez. Play politikası: emoji/büyük harf yığını, "en iyi / 1 numara" gibi doğrulanamayan iddialar, anahtar kelime tekrarı ve başka oyun adları kullanılmaz.

## Uygulama adı (≤ 30)

```
BUNKER — Son Kalanlar
```

## Kısa açıklama (≤ 80)

```
Sığınakta yer sınırlı. Kartlarını aç, arkadaşlarını ikna et, son kalanlardan ol.
```

## Tam açıklama (≤ 4000)

```
Dünya bir felaketle çöktü. Ayakta kalan son sığınağın kapısı açık ama içeride yalnızca bir ya da iki kişilik yer var. Kim içeri girmeyi hak ediyor?

BUNKER — Son Kalanlar, arkadaşlarınla oynayabileceğin çok oyunculu bir tartışma ve eleme oyunudur. Her oyuncunun elinde, sığınağa neler katabileceğini anlatan gizli karakter kartları vardır. Kartlarını doğru zamanda aç, diğerlerini ikna et, gizli oylamada elenmekten kurtul.

NASIL OYNANIR
• Bir oda kur ve 6 haneli davet kodunu arkadaşlarınla paylaş. 4 ile 10 oyuncu aynı odada oynar.
• Herkese beş kart dağıtılır: meslek, sağlık durumu, hobi ya da beceri, yanında getirdiği eşya ve gizli bir sır.
• İlk beş turda her turda bir kartını açarsın. İlk turda mesleğini, sonra hangi kartı açacağını sen seçersin.
• Yazılı sohbette tartışın: Kim sığınak için gerçekten gerekli, kim yük?
• Gizli oylamayla her turda bir oyuncu elenir. Sığınaktaki yer sayısı kadar oyuncu kaldığında oyun biter ve kalanlar kazanır.

ÖZELLİKLER
• Rastgele seçilen felaket senaryoları ve her oyunda yeniden karılan kartlar.
• Oda sahibi sığınak kapasitesini 1 ya da 2 kişi olarak seçer.
• Elenen oyuncular izleyici olarak oyunu sonuna kadar takip eder.
• Bağlantın koparsa oyuna kaldığın yerden dönersin.
• Oyun bitince "Tekrar oyna" ile aynı odada yeni bir tur başlatın.
• Kayıt gerekmez: bir takma ad seç ve oynamaya başla.

GÜVENLİ OYUN
• Sohbette küfür filtresi bulunur.
• Rahatsız eden oyuncuyu kolayca engelleyebilir ya da raporlayabilirsin.
• Oda sahibi lobide istemediği oyuncuyu odadan çıkarabilir.
• Reklam yok, uygulama içi satın alma yok.

Oyun tamamen Türkçedir ve internet bağlantısı gerektirir. Oynamak için en az 4 kişi gerekir; arkadaşlarını topla ve sığınağın kapısında buluşun.
```

## Grafikler

| Varlık | Şart (Play) | Dosya | Durum |
|---|---|---|---|
| Uygulama simgesi | 512×512 PNG, 32 bit, ≤ 1 MB | `apps/mobile/branding/play_icon_512.png` | ✅ |
| Tanıtım görseli | 1024×500 PNG/JPEG, saydamlık yok, ≤ 15 MB | `apps/mobile/branding/feature_graphic_1024x500.png` (`make_icons.py` üretir) | ✅ |
| Telefon ekran görüntüleri | 2–8 adet, PNG/JPEG ≤ 8 MB, kenar 320–3840 px, **uzun kenar kısa kenarın en fazla 2 katı** (ör. 1080×1920; 1080×2400 olmaz). Öne çıkarılma için en az 1080 px önerilir | `docs/store/screenshots/` | ⬜ |

Çekim: `flutter run --release --dart-define-from-file=config/test.json` (release derlemede ortam etiketi görünmez), emülatör 1080×1920 profilinde, `adb exec-out screencap -p > 01-lobi.png` (pencere kesiti değil). Önerilen ekran görüntüleri (gerçek cihazdan, test sunucusuyla, 4 oyunculu bir oyunda; bildirim çubuğu temiz): 1) hoş geldin / takma ad, 2) lobi (oyuncular, davet kodu, kapasite), 3) senaryo tanıtımı, 4) kart açma, 5) tartışma ve sohbet, 6) oylama, 7) sonuç ekranı. Gerçek kişilere ait takma ad veya sohbet görünmemeli; botlar (`cmd/testbot`) ve test adları kullanılır.

## Diğer alanlar

| Alan | Değer |
|---|---|
| Uygulama / oyun | Oyun |
| Kategori | **Kart** (alternatif: Gündelik). Play'de "parti" kategorisi yok; güncel liste doldurma anında kontrol edilir |
| Etiketler | Çok oyunculu, parti, sosyal çıkarım (Play'in listesinden en fazla 5) |
| İletişim e-postası | kalembahriyeli@gmail.com (herkese görünür) |
| Web sitesi | `https://burakkuru5534.github.io/siginak-legal/` |
| Gizlilik politikası | `https://burakkuru5534.github.io/siginak-legal/gizlilik.html` |
