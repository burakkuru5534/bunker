# Yazı tipleri

| Aile | Kaynak | Lisans | Ağırlıklar |
|---|---|---|---|
| Oswald | [google/fonts `ofl/oswald`](https://github.com/google/fonts/tree/main/ofl/oswald) (sürüm 4.103) | SIL OFL 1.1 — `oswald/OFL.txt` | 500, 600, 700 |
| Inter | [google/fonts `ofl/inter`](https://github.com/google/fonts/tree/main/ofl/inter) (sürüm 4.001) | SIL OFL 1.1 — `inter/OFL.txt` | 400, 500, 600 |

Değişken (variable) yazı tiplerinden `fontTools` ile **sabit ağırlıklar** üretildi ve uygulama boyutunu küçültmek için
Latin-1, Latin Extended-A (Türkçe karakterler dahil), genel noktalama ve para birimi (₺) aralıklarına **alt küme** alındı.
Lisanslarda "Reserved Font Name" tanımlı olmadığından değiştirilmiş sürümler aynı adla kullanılabilir.
Lisans metinleri uygulama içindeki lisans sayfasına `LicenseRegistry` ile eklenir (`core/theme/font_licenses.dart`).
