# PostgreSQL migration'ları

`backend/cmd/migrate` ile uygulanır (DB-002). Her ortam (test, prod) kendi şemasına, kendi **migration rolüyle** (`bunker_<ortam>_migrator`) uygulanır; önce test, doğrulandıktan sonra prod (ADR-0009).

## Dosyalar

- Ad: `NNNN_kucuk_harf_ad.sql` (ör. `0001_create_users.sql`). Yeni dosya: `go run ./cmd/migrate new create_users`.
- **Yalnızca ileri yönlü.** Geri alma = yeni migration.
- **Uygulanmış dosya değiştirilmez, silinmez, yeniden adlandırılmaz.** Araç sağlamayı (SHA-256) karşılaştırır; fark varsa çalışmaz.
- Sürümler artan sırayla uygulanır; uygulanmış en büyük sürümden küçük yeni dosya eklenemez.
- Satır sonları LF'ye çevrilerek sağlama hesaplanır (Windows/Linux farkı sorun olmaz).

## Yazım kuralları (`migrate lint` ve CI denetler)

| Yasak | Neden |
|---|---|
| `bunker_test`, `bunker_prod` | Dosyalar şemadan bağımsızdır; hedef şema rolün `search_path`'idir |
| `SET search_path`, `SET ROLE` | Araç yönetir |
| `GRANT`, `REVOKE` | Uygulama rolünün yetkileri her `up` sonunda araç tarafından eşitlenir |
| `CREATE/ALTER/DROP SCHEMA` | Şema `dbsetup` ile kurulur |
| `BEGIN`, `COMMIT`, `ROLLBACK` | Her dosya kendi transaction'ında çalışır; hata olursa tamamı geri alınır |
| `DISABLE ROW LEVEL SECURITY`, `BYPASSRLS` | RLS her tabloda açıktır (ADR-0012) |
| `schema_migrations`, `app_access` | Aracın kendi tablosu ve politikası |

**Transaction dışı:** `CREATE INDEX CONCURRENTLY` gibi ifadeler için dosyanın ilk satırı `-- migrate:no-transaction` olur ve dosya **tek ifade** içerir.

## Her `up` sonunda (ADR-0012)

- Uygulama rolü (`bunker_<ortam>_app`): tüm tablolarda `SELECT, INSERT, UPDATE, DELETE`, dizilerde `USAGE, SELECT`. `TRUNCATE`, DDL ve `schema_migrations` erişimi yok.
- **Salt okunur tablolar:** tablo yorumunda `@app:readonly` varsa uygulama rolü yalnızca `SELECT` alır ve politika `for select` olur. Kataloglar (senaryolar, kartlar) böyledir; içerikleri yalnızca migration'larla değişir:
  ```sql
  comment on table scenarios is '@app:readonly Senaryo kataloğu';
  ```
- Her tabloda RLS açık; yalnızca uygulama rolüne `app_access` politikası. Komutu veya rolü yanlış bir politika bulunursa yeniden kurulur. Supabase'in `anon` / `authenticated` / `service_role` rolleri hiçbir satır göremez.

## Komutlar (`backend/` klasöründe)

```bash
go run ./cmd/migrate lint              # kurallar (DB gerekmez)
go run ./cmd/migrate status            # uygulanmış / bekleyen (test)
go run ./cmd/migrate up -dry-run       # ne uygulanacak
go run ./cmd/migrate up                # uygula (test)
go run ./cmd/migrate up -env prod -yes-prod   # prod (yalnızca test'te doğrulandıktan sonra)
```
