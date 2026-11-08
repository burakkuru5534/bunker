package game

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/burakkuru5534/bunker/backend/internal/game/engine"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
)

// Katalog her başlatmada okunur (~150 satır, indeksli). Başlatma seyrek
// olduğundan önbellek gerekmez; içerik yalnızca migration ile değişir.

// loadCatalog, dağıtım için aktif senaryo kimliklerini ve kartları okur
// (metin okunmaz; yalnızca kimlik ve gizli denge etiketi).
func loadCatalog(ctx context.Context, q persistence.Querier) (engine.Catalog, error) {
	cat := engine.Catalog{Cards: map[engine.Category][]engine.CatalogCard{}}
	rows, err := q.Query(ctx, `select id from scenarios where active order by id`)
	if err != nil {
		return cat, err
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return cat, err
		}
		cat.Scenarios = append(cat.Scenarios, id)
	}
	if err := rows.Err(); err != nil {
		return cat, err
	}
	rows, err = q.Query(ctx, `select id, category, balance from card_definitions where active order by id`)
	if err != nil {
		return cat, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			c        engine.CatalogCard
			category string
		)
		if err := rows.Scan(&c.ID, &category, &c.Balance); err != nil {
			return cat, err
		}
		cat.Cards[engine.Category(category)] = append(cat.Cards[engine.Category(category)], c)
	}
	return cat, rows.Err()
}

// Scenario, istemciye giden senaryo görünümüdür (api.md session.snapshot).
type Scenario struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Disaster string   `json:"disaster"`
	Duration string   `json:"duration"`
	Bunker   []string `json:"bunker"`
	Outside  string   `json:"outside"`
}

// ScenarioOption, lobideki senaryo seçimi listesinin bir satırıdır.
type ScenarioOption struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// loadScenarioOptions, aktif senaryoları başlığa göre sıralı döndürür.
func loadScenarioOptions(ctx context.Context, q persistence.Querier) ([]ScenarioOption, error) {
	rows, err := q.Query(ctx, `select id, title from scenarios where active order by title collate "C", id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (ScenarioOption, error) {
		var o ScenarioOption
		err := row.Scan(&o.ID, &o.Title)
		return o, err
	})
}

func loadScenario(ctx context.Context, q persistence.Querier, id string) (Scenario, error) {
	s := Scenario{ID: id}
	err := q.QueryRow(ctx, `select title, disaster, duration_label, shelter_features, outside_world
		from scenarios where id = $1`, id).Scan(&s.Title, &s.Disaster, &s.Duration, &s.Bunker, &s.Outside)
	return s, err
}

// loadCardTexts, verilen kart kimliklerinin metinleridir.
func loadCardTexts(ctx context.Context, q persistence.Querier, ids []int32) (map[int]string, error) {
	rows, err := q.Query(ctx, `select id, text from card_definitions where id = any($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]string{}
	for rows.Next() {
		var (
			id   int
			text string
		)
		if err := rows.Scan(&id, &text); err != nil {
			return nil, err
		}
		out[id] = text
	}
	return out, rows.Err()
}
