package dbsetup

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Status, bir denetimin sonucudur.
type Status int

const (
	OK Status = iota
	Warn
	Fail
)

func (s Status) Symbol() string { return [...]string{"✓", "!", "✗"}[s] }

// Result, tek denetim sonucudur.
type Result struct {
	Status Status
	Name   string
	Detail string
}

// Check, bağlanmış (ve persistence.Open ile kimliği doğrulanmış) havuz
// üzerinde rol ayarlarını ve yetki sınırlarını denetler.
func Check(ctx context.Context, pool *pgxpool.Pool, schema string, spec RoleSpec) []Result {
	var out []Result
	add := func(st Status, name, format string, args ...any) {
		out = append(out, Result{st, name, fmt.Sprintf(format, args...)})
	}

	var version, user, cur string
	if err := pool.QueryRow(ctx, `select split_part(version(), ' on ', 1), current_user::text, coalesce(current_schema()::text, '')`).
		Scan(&version, &user, &cur); err != nil {
		add(Fail, "sorgu", "%v", err)
		return out
	}
	add(OK, "sunucu", "%s, rol %s, şema %s", version, user, cur)

	// Rol düzeyi sınırlar.
	for _, s := range []struct{ name, want string }{
		{"statement_timeout", spec.StatementTimeout},
		{"lock_timeout", spec.LockTimeout},
		{"idle_in_transaction_session_timeout", spec.IdleInTxTimeout},
	} {
		var got string
		if err := pool.QueryRow(ctx, "select current_setting($1)", s.name).Scan(&got); err != nil {
			add(Fail, s.name, "%v", err)
			continue
		}
		if got == s.want {
			add(OK, s.name, "%s", got)
		} else {
			add(Fail, s.name, "%s (beklenen %s)", got, s.want)
		}
	}

	// Şema yetkileri: uygulama rolü kullanır ama nesne oluşturamaz; migration
	// rolü nesne oluşturur (DDL yalnızca onda).
	var usage, create, dbCreate bool
	if err := pool.QueryRow(ctx, `select has_schema_privilege($1, 'USAGE'), has_schema_privilege($1, 'CREATE'),
		has_database_privilege(current_database(), 'CREATE')`, schema).Scan(&usage, &create, &dbCreate); err != nil {
		add(Fail, "yetki", "%v", err)
		return out
	}
	addBool := func(cond bool, st Status, name, okText, badText string) {
		if cond {
			add(OK, name, "%s", okText)
		} else {
			add(st, name, "%s", badText)
		}
	}
	addBool(usage, Fail, "şema kullanımı", schema+" kullanılabiliyor", schema+" kullanılamıyor (grant usage eksik)")
	if spec.CanCreate() {
		addBool(create, Fail, "şemada DDL", "migration rolü tablo oluşturabilir", "migration rolü şemada nesne oluşturamıyor (grant create eksik)")
	} else {
		addBool(!create, Fail, "şemada DDL", "uygulama rolü tablo oluşturamaz", "uygulama rolü şemada nesne oluşturabiliyor (CREATE kaldırılmalı)")
	}
	addBool(!dbCreate, Warn, "veritabanında DDL", "şema oluşturamaz", "veritabanında şema oluşturabiliyor")

	// Diğer ortamın şeması ve beklenmeyen şemalar.
	rows, err := pool.Query(ctx, `select nspname::text from pg_namespace
		 where has_schema_privilege(oid, 'USAGE')
		   and nspname not in ('pg_catalog', 'information_schema', 'public', $1)
		   and nspname not like 'pg\_%'
		 order by 1`, schema)
	if err != nil {
		add(Fail, "diğer şemalar", "%v", err)
		return out
	}
	var others []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err == nil {
			others = append(others, n)
		}
	}
	rows.Close()
	var leaked []string
	for _, n := range others {
		if strings.HasPrefix(n, "bunker_") {
			leaked = append(leaked, n)
		}
	}
	switch {
	case len(leaked) > 0:
		add(Fail, "ortam yalıtımı", "başka ortamın şemasına erişim var: %s", strings.Join(leaked, ", "))
	case len(others) > 0:
		add(Warn, "diğer şemalar", "PUBLIC üzerinden erişilebilen şemalar: %s (Supabase varsayılanı olabilir)", strings.Join(others, ", "))
	default:
		add(OK, "ortam yalıtımı", "yalnızca %s", schema)
	}

	// Supabase Data API rolleri şemaya erişemez (rol yoksa atlanır).
	for _, r := range []string{"anon", "authenticated", "service_role"} {
		var exists, can bool
		if err := pool.QueryRow(ctx, `select exists(select 1 from pg_roles where rolname = $1)`, r).Scan(&exists); err != nil || !exists {
			continue
		}
		if err := pool.QueryRow(ctx, `select has_schema_privilege($1, $2, 'USAGE')`, r, schema).Scan(&can); err != nil {
			add(Fail, "Data API", "%s: %v", r, err)
			continue
		}
		addBool(!can, Fail, "Data API: "+r, schema+" erişimi yok", schema+" şemasını kullanabiliyor (revoke gerekli)")
	}
	return out
}

// Failed, en az bir Fail var mı?
func Failed(rs []Result) bool {
	for _, r := range rs {
		if r.Status == Fail {
			return true
		}
	}
	return false
}
