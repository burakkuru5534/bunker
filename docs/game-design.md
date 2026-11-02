# Oyun Tasarımı — Sığınak: Son Kalanlar

> Kaynak: GAME-002 … GAME-006 kararları (2026-11-02). Ürün kapsamı için bkz. [product-requirements.md](product-requirements.md).

## 1. Oyuncu sayısı, kapasite ve turlar (GAME-002)

- **Oyuncu sayısı:** 4–10.
- **Sığınak kapasitesi:** Oda sahibi lobide **1 veya 2** seçer. **Varsayılan: 2.** Oyun başladıktan sonra değiştirilemez.
- **Tur modeli (Model A):** Her tur 1 eleme ile biter. `tur sayısı = oyuncu sayısı − kapasite`.
- **Kart açma turları:** Turlar 1–5'te her turda 1 kart açılır. 6. turdan itibaren yeni kart açılmaz; yalnızca tartışma ve oylama yapılır.

| Oyuncu | Kapasite 2 → tur | Kapasite 1 → tur |
|---|---|---|
| 4 | 2 | 3 |
| 5 | 3 | 4 |
| 6 | 4 | 5 |
| 7 | 5 | 6 |
| 8 | 6 | 7 |
| 9 | 7 | 8 |
| 10 | 8 | 9 |

## 2. Oyun aşamaları (GAME-003)

```
LOBBY ──(oda sahibi başlatır; 4–10 oyuncu; herkes hazır)──► DEALING
DEALING ──(kartlar dağıtıldı)──► SCENARIO_INTRO
SCENARIO_INTRO ──(süre doldu)──► ROUND

ROUND döngüsü:
  REVEAL        (yalnızca tur 1–5; oyuncular sırayla 1 kart açar)
  DISCUSSION    (serbest yazılı sohbet)
  VOTING        (gizli oylama)
  [TIE_BREAK]   (yalnızca eşitlik varsa ve izleyici varsa; yoksa kura)
  ELIMINATION   (elenen duyurulur, kartlarının tümü açılır)
  hayatta kalan ≤ kapasite ? ──► FINISHED : sonraki ROUND

FINISHED ──("Tekrar oyna")──► LOBBY (aynı oda, aynı davet kodu)
```

- Tüm geçişleri **sunucu** yapar; istemci yalnızca eylem gönderir.
- Her aşamanın bitiş zamanı (deadline) PostgreSQL'de tutulur. Sunucu yeniden başlarsa süresi dolmuş aşamalar tamamlanır ve oyun kaldığı yerden devam eder.
- **Açıklama sırası:** Oyun başında rastgele belirlenir; her turda başlayan oyuncu bir kişi kayar; elenenler atlanır.

## 3. Kazanma koşulları (GAME-003)

| Durum | Sonuç |
|---|---|
| Hayatta kalan = kapasite | Kalanlar sığınağa girer ve **kazanır**; elenenler kaybeder |
| Ayrılmalar nedeniyle hayatta kalan < kapasite | Kalanlar kazanır; oyun hemen biter |
| Hayatta kimse kalmadı | Oyun **kazanansız** biter, "terk edildi" olarak kaydedilir |

Gizli sır kartlarının kazanmaya etkisi yoktur.

## 4. Ayrılma ve bağlantı kopması (GAME-003)

| Durum | Davranış |
|---|---|
| Bağlantı koptu | **60 sn** bekleme süresi. Bu sürede sırası gelirse otomatik kart açılır; oylamada çekimser sayılır. |
| Bekleme süresi doldu / "Ayrıl" | Oyuncu "ayrıldı" olarak işaretlenir ve **elenmiş sayılır**. Turun elemesi yine oylamayla yapılır. |
| Oylama sırasında aday ayrıldı | O adaya verilen oylar geçersizdir. Hayatta kalan kapasiteye inerse oyun biter. |
| Oda sahibi oyunda ayrıldı | Oyunda özel yetkisi yoktur; oyun sürer, sahiplik devredilir. |
| Ayrılan oyuncu geri döndü | Aynı odaya **yalnızca izleyici** olarak dönebilir; tekrar oyuncu olamaz. |

## 5. İzleyiciler

- Elenen ve ayrılan oyuncular izleyici olur.
- Sohbeti okur, **yazamaz**.
- Açılmış kartları ve elenenlerin tüm kartlarını görür; **hayattaki oyuncuların açılmamış kartlarını göremez**.
- Yalnızca eşitlik bozmada oy kullanır (bkz. §10).

## 6. Kartlar (GAME-004)

Her oyuncuya her kategoriden **1 kart**, toplam **5 kart** verilir.

| Sıra | Kategori | Rolü |
|---|---|---|
| 1 | Meslek | Sığınağa ne katacağı |
| 2 | Sağlık durumu | Güçlü / zayıf yanları |
| 3 | Hobi / beceri | Ek katkı |
| 4 | Eşya | Getirdiği eşya |
| 5 | Gizli sır | Dramatik dönüş kartı |

MVP dışı: aksiyon kartları, biyografi kartları (yaş/cinsiyet), sayısal puan.

### Kart açma kuralı (karma model)

- **1. tur:** Meslek kartı zorunlu.
- **2. turdan itibaren:** Oyuncu açılmamış kartlarından birini serbestçe seçer (gizli sır dahil).
- **Süre dolarsa:** Sunucu, açılmamış kartlardan en düşük sıradakini açar (gizli sır en son).
- Açılan kart geri kapatılamaz. Aynı açma isteği tekrar gelirse yok sayılır (idempotent). İstemci açmadan önce onay sorar.

### Gizlilik

| Kim | Ne görür |
|---|---|
| Kartın sahibi | Kendi 5 kartını her zaman |
| Hayattaki diğer oyuncular | Yalnızca açılmış kartlar |
| İzleyiciler | Açılmış kartlar + elenenlerin tüm kartları |
| Herkes (oyun bitince) | Tüm kartlar |

- Odaya genel yayınlar yalnızca açılmış kartları içerir; oyuncunun kendi kartları yalnızca ona özel mesajla gider.
- Kart kataloğu ve metinleri **sunucuda** tutulur; istemci bir kartın metnini ancak açıldığında (veya sahibiyse) alır.

### Dağıtım

- Aynı oyunda iki oyuncuya **aynı kart verilmez**.
- **Denge etiketi:** Her kartın gizli etiketi vardır (olumlu / nötr / olumsuz). Bir oyuncuya en fazla **2 olumsuz** kart düşer. Etiketler oyunculara gösterilmez.
- Rastgelelik sunucuda, kriptografik olarak güvenli kaynakla sağlanır.
- Hedef: kategori başına ~30 kart (kesin sayı CARD-002…006'da).

### İçerik ilkeleri

- Gerçek kişi isimleri yok.
- Etnik köken, din, siyaset üzerinden aşağılama yok.
- Cinsel içerik, kendine zarar, ayrıntılı şiddet yok.
- Sağlık kartları gerçek hastalık ve engellerle alay etmez; nötr dil kullanılır.
- Gizli sırlar dramatik ama aile dostu (PG-13).

## 7. Senaryolar (GAME-005)

- Her oyunda **rastgele** seçilir; oda sahibi seçemez.
- Senaryonun **mekanik etkisi yoktur**; tartışmanın zeminini oluşturur.
- Gerçek ve yakın tarihli felaketler (deprem, sel, yangın) kullanılmaz.
- Yapı: başlık, felaket (2–3 cümle), kalış süresi, sığınak özellikleri (3 madde), dış dünya.

| # | Başlık | Kalış | Öne çıkan |
|---|---|---|---|
| 1 | Nükleer Kış | 3 yıl | Erzak tam; ısıtma bakım istiyor; tıbbi malzeme az |
| 2 | Kızıl Öksürük | 18 ay | Laboratuvar var ama kullanacak kimse yok; hava filtreleri yedekli |
| 3 | Göktaşı | 2 yıl | Su arıtma var; jeneratör yakıtı sınırlı; teknik kütüphane |
| 4 | Makinelerin Sessizliği | 1 yıl | Elektronik cihaz yasak (konumu ele verir); mekanik aletler var |
| 5 | Güneş Fırtınası | 8 ay | Radyasyon kalkanı sağlam; erzak dar; güneş panelleri hasarlı |

Kesin metinler CARD-007'de yazılır.

## 8. Aşama süreleri (GAME-006)

| Aşama | Süre | Erken bitiş |
|---|---|---|
| SCENARIO_INTRO | 15 sn | — |
| REVEAL (tur 1–5) | Oyuncu başına 20 sn, sırayla | Oyuncu kartını açınca sıra geçer |
| DISCUSSION | Hayatta ≤6 ise 90 sn; ≥7 ise 120 sn | Hayattaki herkes "Hazırım" derse |
| VOTING | 30 sn | Hayattaki herkes oy verince |
| TIE_BREAK | 20 sn | Tüm izleyiciler oy verince |
| ELIMINATION | 8 sn | — |
| FINISHED | Süresiz | Oda sahibi "Tekrar oyna" der |

Süreler MVP'de sabittir. Tahmini en uzun süre (erken bitiş olmadan): 4/2 → ~7 dk, 6/2 → ~15 dk, 10/2 → ~33 dk, 10/1 → ~35 dk.

## 9. Oylama (GAME-006)

- Yalnızca hayattaki oyuncular oy verir.
- **Kendine oy yasak.**
- **Oy değiştirilemez.** Oyuncu başına tur başına 1 oy (veritabanında tekil kısıt).
- Oy vermeyen çekimser sayılır; ayrı "çekimser" düğmesi yoktur.
- Hiç oy verilmezse hayattaki herkes 0 oyla eşit sayılır → eşitlik kuralı.
- Oylama sırasında yalnızca "x/y oy verildi" sayacı gösterilir.
- **Sonuçta** elenen oyuncu ve **adaylara gelen oy sayıları** gösterilir; kimin kime oy verdiği gösterilmez.
- Ayrılan adaya verilen oylar geçersizdir.

## 10. Eşitlik bozma (GAME-001 / GAME-006)

1. Eşitlik yoksa en çok oy alan elenir.
2. Eşitlik var ve **izleyici yoksa** (ör. ilk tur): eşit oy alanlar arasından **kura**.
3. Eşitlik var ve **izleyici varsa**: izleyiciler 20 sn içinde **yalnızca eşit oy alan adaylar arasından** oy verir.
   - Süre içinde oy verenler sayılır. En az 1 oy varsa ve oy verenlerin **hepsi aynı adayı** seçtiyse o aday elenir (tek izleyici varsa kararı o verir).
   - Oylar farklı adaylara dağılırsa veya hiç oy gelmezse → **kura**.
4. **Kura:** Sunucuda kriptografik olarak güvenli rastgele sayıyla, yalnızca eşit oy alan adaylar arasından çekilir. Sonuç ve adaylar kaydedilir; herkese "kura ile belirlendi" olarak duyurulur.
