# ADR-0009: Tek Supabase projesi, test ve prod için şema + rol izolasyonu

- **Durum:** Kabul edildi (2026-11-02)

## Bağlam
Supabase Free planında en fazla 2 aktif proje olabilir; kullanıcının hesabında başka bir proje zaten var. Test ve production yapılandırmalarının yine de ayrılması gerekiyor (MVP kriteri 16). Şimdilik ücretli plana geçilmeyecek.

## Karar
- Tek Supabase projesi (bölge: `eu-central-1` Frankfurt), içinde iki şema:
  - `bunker_test` — test ortamı (Cloud Run `bunker-api-test`)
  - `bunker_prod` — production (Cloud Run `bunker-api`)
- Her şema için ayrı veritabanı rolü: `bunker_test_app`, `bunker_prod_app`. Her rol **yalnızca kendi şemasında** yetkilidir; diğer şemaya erişimi yoktur (`REVOKE` + yalnızca kendi şemasına `GRANT`).
- Her rolün `search_path`'i kendi şemasıdır; migration'lar şemadan bağımsız yazılır ve hedef şemaya uygulanır.
- Her ortamın `DATABASE_URL`'i ayrı secret olarak Secret Manager'da tutulur.
- Migration'lar önce `bunker_test`'e, doğrulandıktan sonra `bunker_prod`'a uygulanır.
- Yerel geliştirme ve otomatik testler Docker / bulut çalışma alanındaki yerel PostgreSQL'de çalışır; Supabase'e dokunmaz.
- Her iki şemada RLS etkin, politika yok; şemalar Data API'ye açılmaz ([security.md](../security.md)).

## Alternatifler
- **İki ayrı Supabase projesi:** En güçlü izolasyon, ancak Free plan limiti (2 proje) dolu; Pro'ya geçiş veya mevcut projeyi duraklatmak gerekir.
- **Test ortamı yalnızca yerel / Docker:** Bulutta test ortamı olmaz; Cloud Run + Supabase bağlantı davranışı prod'dan önce doğrulanamaz.

## Sonuçlar
- Ek maliyet yok.
- Test ve prod **aynı kaynakları paylaşır**: 500 MB depolama, bağlantı limitleri, CPU/RAM. Test yükü prod'u etkileyebilir → test ortamında yük testi yapılmaz (yük testleri yerelde).
- Proje 1 hafta hareketsiz kalırsa her iki ortam birlikte duraklar.
- Yanlış bağlantı dizesi riski rol izolasyonuyla sınırlanır: test rolü prod verisini göremez.
- Ayrı projeye geçiş (Pro planı ile) ileride yalnızca bağlantı dizesi ve migration uygulama değişikliğidir.
