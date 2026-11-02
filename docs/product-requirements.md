# Ürün Gereksinimleri — Sığınak: Son Kalanlar

> Kaynak: GAME-001 kararları (2026-11-02). Oyun kurallarının ayrıntısı için bkz. [game-design.md](game-design.md).

## 1. Ürün özeti

Sığınak: Son Kalanlar, büyük bir felaket sonrasında kapasitesi sınırlı bir sığınağa girmek için
oyuncuların birbirini ikna etmeye çalıştığı, çok oyunculu bir parti oyunudur. Oyuncular karakter
kartlarını açar, yazılı sohbette tartışır ve gizli oyla her turda bir oyuncuyu eler.

- **Pazar:** Türkiye
- **Dil:** Yalnızca Türkçe (metinler baştan l10n/ARB dosyalarında tutulur)
- **Platform:** Android öncelikli; iOS MVP sonrasına ertelendi (yerelde Mac/Xcode yok)
- **Oyuncu sayısı:** 4–10

## 2. MVP kapsamı

| Alan | Kapsam |
|---|---|
| Oturum | Misafir oturumu. Takma ad 3–16 karakter, küfür filtresinden geçer. Avatar yok; baş harf + renk. |
| Oda | Oluşturma, 6 haneli davet koduyla katılma, ayrılma, oda sahipliği devri, boş oda temizliği |
| Lobi | Oyuncu listesi, hazır ol, kapasite seçimi (1 veya 2), oda sahibinin başlatması |
| Oyun | Senaryo, kart dağıtımı, kart açma, tartışma, gizli oylama, eleme, turlar, kazanma, sonuç ekranı |
| Sohbet | Lobi ve oyun içi yazılı sohbet (bkz. §4) |
| Bağlantı | Kopma sonrası yeniden bağlanma ve güncel duruma dönüş |
| Güvenlik | Oyuncu raporlama, engelleme, oda sahibinin lobiden atma yetkisi |
| Kalıcılık | Oyun sonuçları PostgreSQL'e kaydedilir |
| Tekrar | "Tekrar oyna" ile aynı oda ve davet koduyla lobiye dönüş |

## 3. MVP dışı

- Herkese açık oda listesi ve eşleştirme (MVP'de **yalnızca davet kodu**)
- Oyun içinde oyuncu atma (atma **yalnızca lobide**)
- Sesli sohbet
- Arkadaşlık sistemi
- Oyuncu profili, istatistikler, maç geçmişi ekranı
- Özel avatarlar
- Yapay zekâ
- Ödeme, abonelik, kozmetik
- Push bildirimleri
- Kalıcı hesap (e-posta / sosyal giriş)
- iOS mağaza yayını
- Çoklu dil
- Aksiyon kartları, puanlama
- Oda sahibinin süreleri veya senaryoyu seçmesi

## 4. Yazılı sohbet kuralları

| Kural | Karar |
|---|---|
| Açık olduğu aşamalar | Lobi, senaryo, kart açma, tartışma, oylama, eleme, sonuç |
| Elenen oyuncular | İzleyici olarak kalır; sohbeti okur, **yazamaz** |
| Mesaj uzunluğu | En fazla 200 karakter |
| Mesaj sıklığı | Saniyede 1, 10 saniyede en fazla 5 (sunucu tarafında) |
| Küfür filtresi | Sunucu tarafında, Türkçe kelime listesiyle maskeleme |
| Engelleme | Engellenen oyuncunun mesajları engelleyene gösterilmez; oyun kurallarını etkilemez |
| Raporlama | Rapora o odanın son mesajları kanıt olarak eklenir |
| Saklama | Mesajlar PostgreSQL'de tutulur, **30 gün** sonra silinir |

## 5. Mağaza ve içerik

- Yabancılar arası sohbet nedeniyle raporlama ve engelleme zorunludur (Google Play kullanıcı içeriği politikası).
- İçerik aile dostu (PG-13) hedeflenir; kesin yaş derecelendirmesi SAFE-012'de belirlenir.
- Gerçek ve yakın tarihli felaketler (deprem, sel, yangın) senaryo olarak kullanılmaz.

## 6. MVP tamamlanma kriterleri

1. Oyuncu misafir oturumu oluşturabilir.
2. Oyuncu oda oluşturabilir.
3. Diğer oyuncular davet koduyla katılabilir.
4. En az dört oyuncu aynı maça katılabilir.
5. Oyuncular yalnızca kendilerine ait gizli kartları görebilir.
6. Oyun aşamaları Go backend tarafından yönetilir.
7. Oylama ve eleme doğru çalışır.
8. Kazanma koşulu doğru hesaplanır.
9. Oyun sonucu PostgreSQL'e kalıcı olarak kaydedilir.
10. Bağlantısı kesilen oyuncu yeniden bağlanabilir.
11. Bir oyuncu başka bir odanın verilerine erişemez.
12. Geçersiz veya tekrar gönderilen eylemler oyunu bozamaz.
13. Backend yeniden başladığında kalıcı oyun durumu kurtarılabilir.
14. Kritik testler başarılıdır.
15. Android derlemesi alınabilir. *(iOS derlemesi MVP sonrasına ertelendi.)*
16. Test ve production yapılandırmaları ayrılmıştır.
17. Temel güvenlik ve maliyet dokümantasyonu hazırdır.
