-- 0009 room scenario choice
-- Oda sahibi lobide felaket senaryosunu seçebilir (2026-11-08 kullanıcı
-- kararı; önceden her oyunda rastgeleydi). null = rastgele (varsayılan).
-- Seçilen senaryo sonradan pasifleşirse başlatma rastgeleye düşer
-- (internal/game/start.go). Senaryo kataloktan silinirse seçim boşalır.
alter table rooms
  add column scenario_id text references scenarios (id) on delete set null;

-- Yabancı anahtar indeksli olmalı (senaryo silinirken oda taraması olmasın).
create index rooms_scenario_id_idx on rooms (scenario_id);
