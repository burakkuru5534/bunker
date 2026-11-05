# Yasal metinler (SAFE-008)

`site/` klasörü, Google Play'in istediği herkese açık gizlilik politikası ve kullanım koşulları sayfalarıdır (statik HTML, dış kaynak yok).

| Dosya | İçerik |
|---|---|
| `site/gizlilik.html` | Gizlilik Politikası ve KVKK Aydınlatma Metni |
| `site/kosullar.html` | Kullanım Koşulları |
| `site/index.html` | Bağlantı sayfası |

## Yayınlama (GitHub Pages, ücretsiz)

Ana repo özel olduğu sürece Pages ücretsiz planda yayınlanamaz; bu yüzden ayrı, **herkese açık** küçük bir repo kullanılır:

1. GitHub'da herkese açık `siginak-legal` reposunu oluşturun.
2. `site/` içindeki dosyaları o reponun köküne kopyalayıp gönderin.
3. Repo → Settings → Pages → Source: **Deploy from a branch**, Branch: `main` / `(root)`.
4. Adresler: `https://burakkuru5534.github.io/siginak-legal/gizlilik.html` ve `…/kosullar.html` (uygulamadaki varsayılan bağlantılar bunlardır; farklıysa `--dart-define=PRIVACY_URL=… --dart-define=TERMS_URL=…`).
5. Google Play Console → Uygulama içeriği → Gizlilik politikası alanına gizlilik adresini yazın.

## Güncelleme kuralı

Veri işleme değişirse (yeni veri, yeni sağlayıcı — ör. hata raporlama REL-004, saklama süresi) metin **aynı adımda** güncellenir ve yürürlük tarihi değiştirilir. Kaynak tablo: `docs/security.md` §6.

> Bu metinler bir hukukçu tarafından hazırlanmamıştır; yayından önce bir uzmana (özellikle KVKK m.9 yurt dışı aktarım usulü ve aydınlatma yükümlülüğü için) gözden geçirtilmesi önerilir.
