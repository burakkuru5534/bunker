-- 0008 inactive users index (SAFE-014)
-- Kullanılmayan misafir hesaplar (son kullanımdan 90 gün sonra, oturumu
-- kalmamış) bakım işinde anonimleştirilir (internal/maintenance). Kısmi
-- indeks yalnızca silinmemiş hesapları içerir; silinenler bir daha taranmaz.
create index users_active_last_seen_idx on users (last_seen_at) where deleted_at is null;
