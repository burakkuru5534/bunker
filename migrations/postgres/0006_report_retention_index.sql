-- 0006 report retention index (SAFE-009)
-- Sonuçlanmış raporlar (actioned/dismissed) sonuçlandıktan 180 gün sonra
-- toplu silinir (internal/maintenance). İndeks yoksa her parti raporlar
-- tablosunu baştan sona tarar; açık raporlar kuyruk indeksindedir
-- (reports_queue_idx), bu kısmi indeks yalnızca kapanmışları içerir.
create index reports_resolved_at_idx on reports (resolved_at) where status in ('actioned', 'dismissed');
