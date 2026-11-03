# Marka ve Görsel Dil — Sığınak: Son Kalanlar

> Kaynak: UI-001 (2026-11-02). Uygulaması: MOB-008 (tasarım sistemi bileşenleri), UI-003 (tasarım sistemi).

## 1. Marka özü

| | |
|---|---|
| **Ad** | Sığınak: Son Kalanlar (mağaza adı aynı) |
| **Tek cümle** | "Sığınağa yalnızca birkaçınız girebilir. İkna et ya da elen." |
| **Duygu** | Gerilim + parti eğlencesi. Karanlık bir sığınak, uyarı ışıkları, umut veren bir kapı. |
| **Kişilik** | Zeki, iğneleyici ama asla kaba değil; arkadaş grubu oyunu |
| **Kaçınılan** | Kan, gore, gerçek felaket görüntüleri, askeri/şiddet estetiği |

## 2. Tema

- **MVP'de yalnızca koyu tema.** Sığınak atmosferine uyar, parti oyunları çoğunlukla akşam oynanır, tek tema bakım yükünü azaltır.
- Material 3 `ColorScheme` (dark) üzerine kurulur; renkler aşağıdaki token'lardan türetilir.

## 3. Renk paleti

Kontrast oranları WCAG 2.1'e göre hesaplanmıştır (normal metin için en az 4,5:1).

### Temel

| Token | Hex | Kullanım | Arka plan üzerinde kontrast |
|---|---|---|---|
| `background` | `#0F1115` | Ekran arka planı (bunker kömürü) | — |
| `surface` | `#1A1D23` | Kartlar, sohbet balonu | — |
| `surfaceHigh` | `#242832` | Yükseltilmiş yüzey, giriş alanları | — |
| `textPrimary` | `#ECEDEE` | Ana metin | 16,1 |
| `textSecondary` | `#9BA1A6` | Yardımcı metin, zaman damgası | 7,2 |
| `primary` | `#F2A33A` | Uyarı turuncusu: ana düğmeler, sayaç, vurgular | 9,1 |
| `onPrimary` | `#1A1200` | Turuncu düğme üstündeki metin | 8,9 (turuncu üstünde) |
| `secondary` | `#4FB3A9` | Sığınak / umut: kazananlar, "hazır" durumu | 7,5 |
| `danger` | `#F66A6E` | Eleme, hata, oylama hedefi | 6,5 (`surfaceHigh` üzerinde 5,1) |
| `success` | `#4CC38A` | Onay, bağlantı kuruldu | 8,5 |

### Kart kategorileri

Renk tek başına anlam taşımaz: her kategori **ikon + etiket + renk** üçlüsüyle gösterilir.

| Kategori | Renk | Hex | İkon (Material Symbols) | Kontrast (bg) |
|---|---|---|---|---|
| Meslek | Mavi | `#6E9BF5` | `work` | 6,9 |
| Sağlık durumu | Pembe | `#F27A93` | `medical_services` | 7,2 |
| Hobi / beceri | Mor | `#A98BF5` | `palette` | 6,9 |
| Eşya | Yeşil | `#7CC46A` | `backpack` | 9,0 |
| Gizli sır | Altın | `#E0C15A` | `lock` (açılınca `lock_open`) | 10,8 |

## 4. Tipografi

Her iki yazı tipi de Türkçe karakterleri (ç, ğ, ı, İ, ö, ş, ü) destekler ve SIL Open Font License ile ücretsizdir. Yazı tipleri **uygulamaya gömülür** (çalışma anında indirme yok → çevrimdışı çalışır, ek paket gerekmez).

| Rol | Yazı tipi | Ağırlık | Kullanım |
|---|---|---|---|
| Başlık / gösterim | **Oswald** | 500–700 | Ekran başlıkları, sayaç, "ELENDİN", senaryo başlığı |
| Gövde / arayüz | **Inter** | 400–600 | Metin, düğmeler, sohbet, kart metni |

| Stil | Font | Boyut / satır | |
|---|---|---|---|
| `displayLarge` | Oswald 700 | 40 / 48 | Sonuç ekranı |
| `headlineMedium` | Oswald 600 | 28 / 36 | Ekran başlığı |
| `titleLarge` | Oswald 500 | 22 / 28 | Kart başlığı, aşama adı |
| `bodyLarge` | Inter 400 | 16 / 24 | Kart metni |
| `bodyMedium` | Inter 400 | 14 / 20 | Sohbet, açıklamalar |
| `labelLarge` | Inter 600 | 14 / 20 | Düğmeler |
| `labelSmall` | Inter 500 | 11 / 16 | Zaman damgası, rozet |

Büyük harf dönüşümünde Türkçe yerel ayar kullanılır (`i → İ`, `ı → I`).

## 5. İkonografi

- Flutter'ın yerleşik **Material Symbols (Rounded)** seti; ek ikon paketi yok.
- Oyuncu avatarı: baş harf + odadaki sırasına göre sabit renk (8 renklik döngü: kategori ve vurgu renkleri; mockup'larla aynı). Üzerindeki baş harf `background` rengindedir (kontrast ≥ 4,5:1, testli).
- Logo / uygulama ikonu: MVP'de Oswald ile yazılmış "SIĞINAK" kelime işareti + turuncu kapı/ışık sembolü. Nihai ikon ve mağaza görselleri REL-001'de.

## 6. Ses tonu (Türkçe metinler)

- **"Sen" hitabı**, kısa ve doğrudan cümleler.
- Gerilim kur, ama korkutma; espri yap, ama kimseyi aşağılama.
- Hata mesajları suçlamaz, çözüm söyler.

| Durum | Örnek |
|---|---|
| Ana ekran | "Sığınağa kim girecek?" |
| Kart açma sırası | "Sıra sende. Bir kartını aç." |
| Tartışma | "İkna etme zamanı. 90 saniye." |
| Oylama | "Kim dışarıda kalmalı? Oyun gizli." |
| Elenme | "Kapı kapandı. Artık izleyicisin." |
| Kazanma | "Sığınağa girdin. Hayatta kaldın." |
| Bağlantı koptu | "Bağlantı koptu, yeniden bağlanıyoruz…" |
| Oda dolu | "Bu oda dolu. Başka bir kod dene." |

## 7. Hareket (animasyon)

- Kart açma: 300 ms çevirme animasyonu.
- Eleme: kısa kırmızı vurgu + oyuncu kartının solması.
- Son 10 saniye: sayaç turuncudan kırmızıya geçer, hafif titreşim (haptic).
- Sistemde "animasyonları azalt" açıksa çevirme ve titreşim kapatılır, yalnızca solma kalır.

## 8. Erişilebilirlik kuralları

- Metin kontrastı en az 4,5:1 (yukarıdaki tüm token'lar sağlar).
- Dokunma alanları en az 48×48 dp.
- Renk tek başına bilgi taşımaz (ikon + metin).
- Sistem yazı tipi ölçeği %200'e kadar düzen bozulmadan desteklenir (PLAY-012'de test).
- Ekran okuyucu etiketleri tüm ikon düğmelerde zorunlu.
