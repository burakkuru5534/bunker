-- 0005 cleanup indexes (DB-004)
-- Sorgu planı testlerinin (internal/migrate/query_plans_integration_test.go)
-- bulduğu eksikler. Tablolar henüz küçük olduğundan transaction içinde
-- oluşturulur; büyük tablolarda ileride CONCURRENTLY + no-transaction kullanılır.

-- ROOM-010: uzun süre hareketsiz kalan açık odaları bulma.
create index rooms_open_updated_at_idx on rooms (updated_at) where status <> 'closed';

-- ROOM-010: süresi geçmiş kapalı odaları toplu silme.
create index rooms_closed_at_idx on rooms (closed_at) where status = 'closed';

-- Oda silinince reports.room_id boşaltılır (on delete set null); indeks
-- olmadan her oda silmesi raporlar tablosunu baştan sona tarar.
create index reports_room_id_idx on reports (room_id);
