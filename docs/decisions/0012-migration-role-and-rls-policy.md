# ADR-0012: Migration rolü, yetki eşitleme ve RLS politikası

- **Durum:** Kabul edildi (2026-11-03)

## Bağlam
security.md "tüm tablolarda RLS etkin, politika yok" diyordu. Ancak uygulama rolü (`bunker_<ortam>_app`) tabloların sahibi değildir ve `BYPASSRLS` yetkisi yoktur (bilinçli olarak; DB-001a açılışta bunu reddeder). PostgreSQL'de RLS açık ve politika yoksa sahip olmayan her rol **hiçbir satırı göremez**. Bu kural uygulanınca uygulamanın kendisi de çalışamazdı. Ayrıca migration dosyaları şemadan ve ortamdan bağımsız olmalıdır (ADR-0009); rol adları ise ortama göre değişir (`bunker_test_app` / `bunker_prod_app`), bu yüzden yetkiler dosyalarda yazılamaz.

## Karar
- Ortam başına ayrı **migration rolü** `bunker_<ortam>_migrator`: şemada `USAGE, CREATE`; tabloların sahibi olur. `statement_timeout 1min`, `lock_timeout 5s`, bağlantı sınırı 2. Uygulama rolü DDL yapamaz.
- `cmd/migrate up` her çalıştırmanın sonunda (idempotent) **yetkileri eşitler**:
  - Uygulama rolü: tüm tablolarda `SELECT, INSERT, UPDATE, DELETE`, dizilerde `USAGE, SELECT`, gelecekteki nesneler için varsayılan yetkiler. `TRUNCATE`, `REFERENCES`, `TRIGGER`, DDL yok.
  - Her tabloda RLS açık. Yalnızca uygulama rolüne izin veren tek politika `app_access` (`using (true) with check (true)`).
  - `anon`, `authenticated`, `service_role` ve `PUBLIC` için yetki ve politika yoktur, yani hiçbir satır göremezler.
  - `schema_migrations` uygulama rolüne kapalıdır (RLS açık, politika yok).
- Migration dosyalarında `GRANT/REVOKE`, şema adı, `search_path`, `DISABLE ROW LEVEL SECURITY` yasaktır (`migrate lint`, CI).

## Alternatifler
- **Uygulama rolüne `BYPASSRLS`:** RLS'i tamamen etkisiz kılar; bağlantı ele geçirilirse ikinci savunma hattı kalmaz.
- **Tabloların sahibi uygulama rolü olsun:** Uygulama rolü DDL yapabilir hale gelir (DROP TABLE vb.); en az yetki ilkesine aykırı.
- **Kullanıcı bazlı RLS politikaları (`auth.uid()`):** Supabase Auth ve Data API için tasarlanmış; backend tek rolle bağlandığından ve yetki kontrolü uygulama katmanında yapıldığından (SAFE-003) uygulanamaz.
- **Yetkileri her migration dosyasında yazmak:** Ortam adlarını dosyalara sokar (ADR-0009'a aykırı) ve unutulmaya açıktır.

## Sonuçlar
- RLS, Supabase'in Data API rollerine karşı "varsayılan reddet" katmanı olarak kalır; uygulama yetkisi tek bir yerde (migrate) tanımlıdır.
- Elle yapılan yetki/RLS sapmaları bir sonraki `migrate up` ile düzelir.
- Satır düzeyinde kullanıcı yetkilendirmesi veritabanında değil, uygulama katmanındadır (oda üyeliği vb.).
