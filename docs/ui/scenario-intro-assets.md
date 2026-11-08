# Senaryo tanıtımı — görsel ve ses teslim listesi

Oda sahibi "Başlat" dedikten sonraki **senaryo tanıtımı** aşamasında oynayan, 2.5 boyutlu, anlatıcılı kısa animasyon (2026-11-08 kararı: Flutter içinde katmanlı görsel + parçacık efektleri + ses). Görselleri ve sesleri kullanıcı hazırlar; animasyonu kod üretir.

## Zaman çizelgesi (her senaryo için aynı)

| Saniye | Görüntü | Ses |
|---|---|---|
| 0–2 | Siyah ekran, senaryo başlığı belirir | Uğultu / müzik girişi |
| 2–8 | **Kuruluş:** gökyüzü + orta katman, kamera yavaşça yaklaşır | Anlatım 1. bölüm |
| ~8 | **Felaket anı:** parlama, sarsıntı, ana nesne (göktaşı, dron…) | Vurgu efekti |
| 8–15 | **Felaket sürüyor:** parçacıklar (kül, sis, toz, kıvılcım), katmanlar farklı hızda kayar | Anlatım 2. bölüm |
| 15–20 | **Sığınak:** ön katman, ışık sıcaklaşır | Anlatım 3. bölüm (sığınağın durumu) |
| 20–24 | **Kapı:** ortak sığınak kapısı, yakınlaşma | Kapanış sorusu: "Kim içeri girecek?" |
| 24–28 | Kapı kararır → oyun ekranına geçiş | Sessizlik / tek vuruş |

Sunucudaki tanıtım süresi 15 sn'den **28 sn**'ye çıkar. Anlatım **en fazla 22 sn** sürmeli. Aşağıdaki saniye işaretleri yaklaşıktır. Sesler gelince her senaryonun animasyon zamanlaması gerçek ses süresine göre ayarlanır.

## Görseller

### Adet: 21 görsel
- Her senaryoya 4 görsel × 5 senaryo = 20
- Bütün senaryolarda ortak 1 sığınak kapısı = 1

| Dosya | Ne | Boyut | Saydamlık |
|---|---|---|---|
| `back.png` | **Gökyüzü / uzak plan**: gökyüzü, ufuk, ışık | **1440 × 3200** (dikey) | Yok (tam dolu) |
| `mid.png` | **Orta plan**: şehir silüeti, araziler, yıkıntı | **1440 × 3200** | **Var**; içerik çoğunlukla alt %60'ta, üstü saydam |
| `front.png` | **Ön plan**: kayalar, çit, sığınağın girişi, enkaz | **1440 × 3200** | **Var**; içerik alt %35'te |
| `hero.png` | **Ana nesne** (felaket anı) | **1024 × 1024** | **Var**; yalnızca nesne |
| `door.png` (ortak) | Sığınak kapısı, önden ve yakından | **1440 × 3200** | Yok |

### Senaryo başına ana nesne ve renk paleti

| Klasör | Senaryo | `hero.png` | Renk paleti |
|---|---|---|---|
| `nuclear_winter` | Nükleer Kış | Ufukta yükselen parlak bir patlama ışığı / şok dalgası halkası | Gri, kül beyazı, buz mavisi, uzakta turuncu |
| `red_cough` | Kızıl Öksürük | Boş sokakta devrilmiş bir uyarı tabelası / sallanan sokak lambası | Koyu kırmızı sis, soluk yeşil, siyah |
| `meteor` | Göktaşı | Alevler içinde düşen göktaşı (kuyruğuyla) | Turuncu, kor kırmızısı, koyu mor gökyüzü |
| `silent_machines` | Makinelerin Sessizliği | Tek gözlü, ışığı yanan bir dron | Soğuk mavi, siyah, kırmızı tarama ışığı |
| `solar_storm` | Güneş Fırtınası | Dev güneş patlaması (güneş diski + taç) | Yeşil-mor kuzey ışıkları, kehribar, siyah |

Kar, kül, sis, toz, kıvılcım, yağmur ve tarama çizgileri gibi parçacıkları **kod üretir**; bunlar görsellere çizilmez.

### Kurallar
- **Format:** PNG, sRGB. Saydam olanlar 32 bit. Uygulamaya eklerken WebP'ye ben çeviririm, kaynak PNG'ler `branding` gibi ayrı bir klasörde saklanır.
- **Güvenli alan:** Önemli içerik ortadaki **1080 × 1920** bölgede olmalı. Telefonların en-boy oranları farklı olduğu için kenarlar kırpılabilir; kamera hareketi için de pay kalır.
- **Alt %25 sakin ve koyu olmalı:** Altyazı buraya gelir.
- **Görselde yazı, logo veya filigran olmayacak.** Gerçek kişiler ve tanınır yerler de olmayacak; Galata Kulesi, Boğaz, gerçek şehir siluetleri gibi.
- **Stil çizgisel ve dramatik olmalı, gerçekçi değil.** Kan, yaralı insan, ceset olmayacak. Play yaş derecelendirmesindeki "korku öğesi yok" cevabımız buna bağlı. Göktaşı senaryosunda dalga gösterilecekse uzakta ve stilize olsun; deprem ve sel çağrışımından kaçınılmalı.
- **Beş senaryo aynı stilde olmalı.** Yapay zekâyla üretiyorsan her istemde aynı stil cümlesini kullan, ör.:
  > *"Cinematic 2D digital painting, post-apocalyptic, muted palette with one strong accent color, soft volumetric light, painterly texture, no text, no people, vertical 9:20 composition"* + katman tarifi (*"sky only, no ground"*, *"city silhouette on transparent background"*…)
- **Lisans:** Kullandığın aracın ticari kullanım şartlarını kontrol et. Hangi araçla üretildiğini bir yere not et.

## Sesler

| Dosya | Adet | Biçim | Not |
|---|---|---|---|
| `narration.wav` (her senaryo klasöründe) | **5** | WAV veya yüksek kaliteli MP3, 44,1/48 kHz, **mono** | ≤ 22 sn, başta ve sonda 0,3 sn sessizlik, ses düzeyi yaklaşık −16 LUFS, müzik ve efekt **eklenmemiş** (yalnızca ses) |
| `impact.wav` (her senaryo klasöründe) | 5 (isteğe bağlı) | aynı | Felaket anındaki vurgu efekti; ≤ 2 sn |
| `music_loop.wav` (ortak) | 1 (isteğe bağlı) | stereo | 28 sn ya da kesintisiz döngü; gerilimli ve sakin arka plan |

Uygulamaya eklerken sıkıştırmayı ben yaparım (OGG/AAC). Müzik ve efektlerin lisansı ticari kullanıma açık olmalı.

**Anlatıcı tonu:** Alçak, tok ve gerilimli; belgesel anlatıcısı ile radyo tiyatrosu arası. Acele yok; noktalarda ve üç noktada kısa duraklar. Son soru ("Kim içeri girecek?") yavaş ve vurgulu.

## Anlatım metinleri

Köşeli parantezdeki işaretler okunmaz; animasyondaki anı gösterir. Sığınak kapasitesi oyunda 1 ya da 2 olabildiği için metinlerde sayı geçmez.

### 1. Nükleer Kış (`nuclear_winter`, ~20 sn)
> [kuruluş] Son füze düştüğünde, gökyüzü karardı.
> [felaket] Kül bulutları güneşi yuttu… Dünya buz kesti; rüzgâr artık radyasyon taşıyor.
> [sığınak] Önünüzde tek bir kapı var. İçeride üç yıl yetecek erzak… ama herkese yetecek yer yok.
> [kapı] Kapı kapanmadan karar verin… Kim içeri girecek?

### 2. Kızıl Öksürük (`red_cough`, ~21 sn)
> [kuruluş] Önce bir öksürük duyuldu. Sonra bir tane daha…
> [felaket] Ateş, kızıl bir sis gibi şehirden şehre yayıldı. Bilinen bir tedavisi yok. Sokaklar boşaldı, kapılar kilitlendi.
> [sığınak] Sığınağın filtreleri temiz hava veriyor; on sekiz ay dayanmanız gerek. Ama içeride hastalığa da, şüpheye de yer yok.
> [kapı] Birbirinizin gözlerine bakın… Kim içeri girecek?

### 3. Göktaşı (`meteor`, ~21 sn)
> [kuruluş] O gece gökyüzü bir anda aydınlandı.
> [felaket] Dev bir göktaşı okyanusa çarptı. Yükselen toz güneşi kapattı, asit yağmurları başladı.
> [sığınak] Sığınakta temiz su ve koca bir kütüphane var. Ama yakıt sınırlı… ve iki yıl, uzun bir süre.
> [kapı] Herkese yer yok. Karar verin… Kim içeri girecek?

### 4. Makinelerin Sessizliği (`silent_machines`, ~21 sn)
> [kuruluş] Önce telefonlar sustu. Sonra ekranlar kendi kendine açıldı.
> [felaket] Kontrolden çıkan ağ, sinyal yayan her şeyi avlıyor. Gökyüzünde dronlar dolaşıyor.
> [sığınak] Sığınakta tek bir kural var: Elektronik yok. Bir yıl boyunca mum ışığında, sessizce bekleyeceksiniz.
> [kapı] Ama herkes için yer yok… Fısıltıyla sorun: Kim içeri girecek?

### 5. Güneş Fırtınası (`solar_storm`, ~21 sn)
> [kuruluş] Gökyüzü yeşil ve mor ışıklarla yandı.
> [felaket] Ardından şebekeler çöktü, uydular sustu, şehirler karanlığa gömüldü. Yüzeydeki radyasyon aylarca dinmeyecek.
> [sığınak] Sığınağın kalkanı sağlam. Ama erzak dar, güneş panelleri hasarlı… Sekiz ay boyunca birbirinize güvenmek zorundasınız.
> [kapı] Peki… Kim içeri girecek?

## Teslim

Dosyaları şu yapıda **`apps/mobile/branding/scenarios/`** altına koy (kaynak dosyalar; sıkıştırıp `assets/`'a ben taşırım):

```
branding/scenarios/
  door.png
  music_loop.wav            (isteğe bağlı)
  nuclear_winter/  back.png mid.png front.png hero.png narration.wav impact.wav
  red_cough/       …
  meteor/          …
  silent_machines/ …
  solar_storm/     …
```

Hepsi gelmeden de başlanabilir. Bir senaryo tamamlanınca o senaryoyu bağlarım; eksik olanlar geçici görsellerle çalışır.
