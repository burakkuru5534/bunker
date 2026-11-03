package migrate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TrackingTable, uygulanmış migration'ların tutulduğu tablodur (hedef şemada).
const TrackingTable = "schema_migrations"

// PolicyName, uygulama rolüne verilen RLS politikasının adıdır (ADR-0012).
const PolicyName = "app_access"

// Options, hedef şema ve uygulama rolüdür.
type Options struct {
	Schema  string
	AppRole string
	Logger  *slog.Logger
	// LockWait, başka bir migration çalışıyorsa kilit için en çok bekleme
	// (varsayılan 30 sn).
	LockWait time.Duration
}

// Applied, veritabanındaki kayıttır.
type Applied struct {
	Version    int
	Name       string
	Checksum   string
	AppliedAt  time.Time
	DurationMS int
}

// Status, uygulanmış ve bekleyen migration'lardır.
type Status struct {
	Applied []Applied
	Pending []Migration
}

// SyncReport, yetki eşitlemesinin özetidir.
type SyncReport struct {
	Tables          int
	RLSEnabled      []string
	PoliciesCreated []string
}

func (o Options) table() string { return pgx.Identifier{o.Schema, TrackingTable}.Sanitize() }

func (o Options) logger() *slog.Logger {
	if o.Logger != nil {
		return o.Logger
	}
	return slog.Default()
}

// GetStatus, kilit almadan durumu okur ve dosyalarla tutarlılığı denetler.
// Sürüm tablosu yoksa her şey bekliyor sayılır.
func GetStatus(ctx context.Context, pool *pgxpool.Pool, opts Options, migs []Migration) (Status, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return Status{}, err
	}
	defer conn.Release()
	return status(ctx, conn.Conn(), opts, migs)
}

func status(ctx context.Context, conn *pgx.Conn, opts Options, migs []Migration) (Status, error) {
	var exists bool
	if err := conn.QueryRow(ctx, `select to_regclass($1) is not null`, opts.table()).Scan(&exists); err != nil {
		return Status{}, err
	}
	var applied []Applied
	if exists {
		rows, err := conn.Query(ctx, `select version, name, checksum, applied_at, duration_ms from `+opts.table()+` order by version`)
		if err != nil {
			return Status{}, err
		}
		applied, err = pgx.CollectRows(rows, func(r pgx.CollectableRow) (Applied, error) {
			var a Applied
			err := r.Scan(&a.Version, &a.Name, &a.Checksum, &a.AppliedAt, &a.DurationMS)
			return a, err
		})
		if err != nil {
			return Status{}, err
		}
	}
	return plan(applied, migs)
}

// plan, kayıtları dosyalarla karşılaştırır:
//   - uygulanmış her sürümün dosyası var ve adı/sağlaması aynı olmalı
//   - bekleyen sürümler uygulanmış en büyük sürümden büyük olmalı (sıra dışı yok)
func plan(applied []Applied, migs []Migration) (Status, error) {
	byVersion := make(map[int]Migration, len(migs))
	for _, m := range migs {
		byVersion[m.Version] = m
	}
	var errs []error
	maxApplied := 0
	done := map[int]bool{}
	for _, a := range applied {
		done[a.Version] = true
		maxApplied = max(maxApplied, a.Version)
		m, ok := byVersion[a.Version]
		switch {
		case !ok:
			errs = append(errs, fmt.Errorf("%04d_%s uygulanmış ama dosyası yok (dosya silinmemeli)", a.Version, a.Name))
		case m.Name != a.Name:
			errs = append(errs, fmt.Errorf("%04d: uygulanan ad %q, dosyadaki %q (uygulanmış dosya yeniden adlandırılmamalı)", a.Version, a.Name, m.Name))
		case m.Checksum != a.Checksum:
			errs = append(errs, fmt.Errorf("%s uygulandıktan sonra değiştirilmiş (sağlama farklı); değişikliği geri alıp yeni migration ekleyin", m.File))
		}
	}
	var pending []Migration
	for _, m := range migs {
		if done[m.Version] {
			continue
		}
		if m.Version < maxApplied {
			errs = append(errs, fmt.Errorf("%s sıra dışı: %04d uygulanmışken daha küçük sürüm eklenemez; dosyayı yeniden numaralandırın", m.File, maxApplied))
			continue
		}
		pending = append(pending, m)
	}
	if len(errs) > 0 {
		return Status{}, errors.Join(errs...)
	}
	return Status{Applied: applied, Pending: pending}, nil
}

// Up, bekleyen migration'ları sırayla uygular ve ardından uygulama rolünün
// yetkilerini eşitler. dryRun ise yalnızca bekleyenleri döndürür (DB'ye
// yazmaz, sürüm tablosunu da oluşturmaz).
func Up(ctx context.Context, pool *pgxpool.Pool, opts Options, migs []Migration, dryRun bool) ([]Migration, SyncReport, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, SyncReport{}, err
	}
	defer conn.Release()
	c := conn.Conn()

	if dryRun {
		st, err := status(ctx, c, opts, migs)
		return st.Pending, SyncReport{}, err
	}

	unlock, err := lock(ctx, c, opts)
	if err != nil {
		return nil, SyncReport{}, err
	}
	defer unlock()

	if _, err := c.Exec(ctx, `create table if not exists `+opts.table()+` (
		version     integer     primary key,
		name        text        not null,
		checksum    text        not null,
		applied_at  timestamptz not null default now(),
		duration_ms integer     not null
	)`); err != nil {
		return nil, SyncReport{}, fmt.Errorf("sürüm tablosu: %w", err)
	}

	st, err := status(ctx, c, opts, migs)
	if err != nil {
		return nil, SyncReport{}, err
	}
	var applied []Migration
	for _, m := range st.Pending {
		start := time.Now()
		if err := apply(ctx, c, opts, m, start); err != nil {
			return applied, SyncReport{}, err
		}
		applied = append(applied, m)
		opts.logger().Info("migration uygulandı", "version", m.Version, "name", m.Name, "duration_ms", time.Since(start).Milliseconds())
	}

	report, err := SyncPrivileges(ctx, c, opts)
	if err != nil {
		return applied, report, fmt.Errorf("yetki eşitleme: %w", err)
	}
	return applied, report, nil
}

func apply(ctx context.Context, c *pgx.Conn, opts Options, m Migration, start time.Time) error {
	record := func(q interface {
		Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	}) error {
		_, err := q.Exec(ctx, `insert into `+opts.table()+` (version, name, checksum, duration_ms) values ($1, $2, $3, $4)`,
			m.Version, m.Name, m.Checksum, int(time.Since(start).Milliseconds()))
		return err
	}

	if m.NoTx {
		// Tek ifade; rolün search_path'i hedef şemadır (açılışta doğrulandı).
		if _, err := c.Exec(ctx, m.SQL); err != nil {
			return migrationError(m, err)
		}
		if err := record(c); err != nil {
			return fmt.Errorf("%s uygulandı ama kaydedilemedi: %w", m.File, err)
		}
		return nil
	}

	tx, err := c.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(ctx, "set local search_path to "+pgx.Identifier{opts.Schema}.Sanitize()); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, m.SQL); err != nil {
		return migrationError(m, err)
	}
	if err := record(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// migrationError, PostgreSQL hatasını dosya ve satır bilgisiyle açıklar.
func migrationError(m Migration, err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return fmt.Errorf("%s: %w", m.File, err)
	}
	msg := fmt.Sprintf("%s: %s (SQLSTATE %s)", m.File, pgErr.Message, pgErr.Code)
	if pgErr.Position > 0 && int(pgErr.Position) <= len(m.SQL) {
		line := strings.Count(m.SQL[:pgErr.Position-1], "\n") + 1
		msg += fmt.Sprintf(", satır %d", line)
	}
	if pgErr.Detail != "" {
		msg += "; " + pgErr.Detail
	}
	if pgErr.Hint != "" {
		msg += "; ipucu: " + pgErr.Hint
	}
	return errors.New(msg + " — migration geri alındı, kayıt yazılmadı")
}

// lock, şemaya özgü oturum düzeyi advisory lock alır (session mode'da
// güvenli). Başka bir migration çalışıyorsa LockWait boyunca bekler.
func lock(ctx context.Context, c *pgx.Conn, opts Options) (func(), error) {
	key := "siginak.migrate." + opts.Schema
	wait := opts.LockWait
	if wait <= 0 {
		wait = 30 * time.Second
	}
	deadline := time.Now().Add(wait)
	for {
		var ok bool
		if err := c.QueryRow(ctx, `select pg_try_advisory_lock(hashtextextended($1, 0))`, key).Scan(&ok); err != nil {
			return nil, fmt.Errorf("kilit: %w", err)
		}
		if ok {
			return func() {
				uctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_, _ = c.Exec(uctx, `select pg_advisory_unlock(hashtextextended($1, 0))`, key)
			}, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("başka bir migration çalışıyor (%s kilidi %s içinde alınamadı)", opts.Schema, wait)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

// SyncPrivileges, uygulama rolünün yetkilerini ve RLS'i eşitler (ADR-0012).
// Her çalıştırmada aynı sonucu verir (idempotent):
//   - Uygulama rolü: tablolarda SELECT/INSERT/UPDATE/DELETE, dizilerde
//     USAGE/SELECT; ileride oluşacaklar için varsayılan yetkiler. TRUNCATE,
//     REFERENCES, TRIGGER ve DDL yok.
//   - Her tabloda RLS açık; uygulama rolüne tek politika (app_access).
//     anon/authenticated/service_role için politika yoktur → erişemez.
//   - schema_migrations: uygulama rolü erişemez, RLS açık, politika yok.
func SyncPrivileges(ctx context.Context, c *pgx.Conn, opts Options) (SyncReport, error) {
	var rep SyncReport
	s := pgx.Identifier{opts.Schema}.Sanitize()
	app := pgx.Identifier{opts.AppRole}.Sanitize()

	tx, err := c.Begin(ctx)
	if err != nil {
		return rep, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	for _, q := range []string{
		`alter default privileges in schema ` + s + ` grant select, insert, update, delete on tables to ` + app,
		`alter default privileges in schema ` + s + ` grant usage, select on sequences to ` + app,
		`grant select, insert, update, delete on all tables in schema ` + s + ` to ` + app,
		`grant usage, select on all sequences in schema ` + s + ` to ` + app,
		`revoke all on ` + opts.table() + ` from ` + app,
	} {
		if _, err := tx.Exec(ctx, q); err != nil {
			return rep, fmt.Errorf("%s: %w", strings.SplitN(q, " to ", 2)[0], err)
		}
	}

	rows, err := tx.Query(ctx, `
		select c.relname::text, c.relrowsecurity,
		       exists (select 1 from pg_catalog.pg_policies p
		                where p.schemaname = n.nspname and p.tablename = c.relname and p.policyname = $2)
		  from pg_catalog.pg_class c
		  join pg_catalog.pg_namespace n on n.oid = c.relnamespace
		 where n.nspname = $1 and c.relkind in ('r', 'p')
		 order by 1`, opts.Schema, PolicyName)
	if err != nil {
		return rep, err
	}
	type tbl struct {
		name     string
		rls, pol bool
	}
	tables, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (tbl, error) {
		var t tbl
		err := r.Scan(&t.name, &t.rls, &t.pol)
		return t, err
	})
	if err != nil {
		return rep, err
	}

	for _, t := range tables {
		rep.Tables++
		qualified := pgx.Identifier{opts.Schema, t.name}.Sanitize()
		if !t.rls {
			if _, err := tx.Exec(ctx, `alter table `+qualified+` enable row level security`); err != nil {
				return rep, fmt.Errorf("%s RLS: %w", t.name, err)
			}
			rep.RLSEnabled = append(rep.RLSEnabled, t.name)
		}
		if t.name == TrackingTable || t.pol {
			continue
		}
		if _, err := tx.Exec(ctx, `create policy `+PolicyName+` on `+qualified+
			` as permissive for all to `+app+` using (true) with check (true)`); err != nil {
			return rep, fmt.Errorf("%s politika: %w", t.name, err)
		}
		rep.PoliciesCreated = append(rep.PoliciesCreated, t.name)
	}
	return rep, tx.Commit(ctx)
}
