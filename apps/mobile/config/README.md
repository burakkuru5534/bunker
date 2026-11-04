# Ortam yapılandırmaları

Uygulama, derleme zamanında `--dart-define-from-file` ile yapılandırılır. Bu dosyalar **gizli değer içermez**; yalnızca herkese açık API adresleri bulunur.

| Dosya | Ortam | Git |
|---|---|---|
| `local.json` | Yerel backend (kişiye özel; ör. telefonda bilgisayarın yerel IP'si) | **Dışlanır** — `local.example.json`'dan kopyalanır |
| `test.json` | Cloud Run `bunker-api-test` | İzlenir |
| `prod.json` | Cloud Run `bunker-api` | İzlenir |

> `test.json` ve `prod.json` içindeki `.example.invalid` adresleri yer tutucudur; Cloud Run servisleri oluşturulunca gerçek adreslerle değiştirilir.

## Yerel kurulum

```bash
cp config/local.example.json config/local.json
```

- **Android emülatörü:** `10.0.2.2` bilgisayarın `localhost`'udur — örnekteki değer olduğu gibi çalışır.
- **Gerçek telefon (aynı Wi‑Fi):** `10.0.2.2` yerine bilgisayarın yerel IP'sini yazın (ör. `http://192.168.1.20:8080`). IP'yi Windows'ta `ipconfig` → "IPv4 Address" satırı verir. Backend'i ilk çalıştırdığınızda Windows Güvenlik Duvarı izin sorarsa yalnızca **Özel ağlar**'a izin verin; telefon ve bilgisayar aynı Wi‑Fi'da olmalı (misafir ağları cihazları birbirinden ayırabilir). Telefonun tarayıcısında `http://<IP>:8080/health` açılıyorsa bağlantı hazırdır.

## Çalıştırma

```bash
flutter run --dart-define-from-file=config/local.json
flutter run --dart-define-from-file=config/test.json
flutter build apk --release --dart-define-from-file=config/prod.json
```

Dosya verilmezse uygulama `local` ortamında emülatör adresleriyle açılır.

## Kurallar

- `test` ve `prod` ortamlarında `https` / `wss` zorunludur; aksi halde uygulama açılışta hata verir.
- Düz `http` / `ws` yalnızca **debug** derlemede izinlidir (`android/app/src/debug/AndroidManifest.xml`).
