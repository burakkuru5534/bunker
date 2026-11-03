package persistence

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultAcquireTimeout, havuzdan bağlantı almak için en uzun bekleme
// (architecture.md §11: "havuz doluysa hızlı hata", ≤ 3 sn).
const DefaultAcquireTimeout = 3 * time.Second

// slowAcquire: bu süreden uzun bekleme uyarı olarak loglanır (havuz baskısı).
const slowAcquire = 500 * time.Millisecond

// ErrPoolExhausted, bağlantı AcquireTimeout içinde alınamadığında döner.
// context.DeadlineExceeded'ı sarar: istek katmanı 500 INTERNAL yazar ve
// WARNING loglar (httpapi.WriteAppError).
var ErrPoolExhausted = fmt.Errorf("veritabanı havuzu dolu: %w", context.DeadlineExceeded)

// Querier, Read ve Tx içinde kullanılan ortak arayüzdür (pgx.Tx ve
// *pgxpool.Conn ikisi de sağlar).
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// DB, uygulamanın veritabanı erişim noktasıdır. Her erişim bağlantıyı
// AcquireTimeout ile sınırlı bekleyerek alır, işi yapar ve bırakır;
// bağlantı hiçbir zaman fonksiyon dışına sızmaz.
type DB struct {
	pool           *pgxpool.Pool
	acquireTimeout time.Duration
	logger         *slog.Logger
}

// NewDB, havuzu sarar. acquireTimeout ≤ 0 ise DefaultAcquireTimeout.
func NewDB(pool *pgxpool.Pool, acquireTimeout time.Duration, logger *slog.Logger) *DB {
	if acquireTimeout <= 0 {
		acquireTimeout = DefaultAcquireTimeout
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &DB{pool: pool, acquireTimeout: acquireTimeout, logger: logger}
}

// Pool, alttaki havuzdur (yalnızca araçlar ve testler için).
func (d *DB) Pool() *pgxpool.Pool { return d.pool }

// Close, havuzu kapatır.
func (d *DB) Close() { d.pool.Close() }

// Ping, hazırlık denetimi içindir (/health/ready).
func (d *DB) Ping(ctx context.Context) error {
	return d.Read(ctx, func(q Querier) error {
		_, err := q.Exec(ctx, "select 1")
		return err
	})
}

// Stats, havuz istatistikleridir (gözlem için).
func (d *DB) Stats() *pgxpool.Stat { return d.pool.Stat() }

func (d *DB) acquire(ctx context.Context) (*pgxpool.Conn, error) {
	actx, cancel := context.WithTimeout(ctx, d.acquireTimeout)
	defer cancel()
	start := time.Now()
	conn, err := d.pool.Acquire(actx)
	waited := time.Since(start)
	if err != nil {
		// İstek bağlamı hâlâ canlıysa süre bizim acquire sınırımızdan doldu.
		if ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
			st := d.pool.Stat()
			d.logger.Warn("veritabanı havuzu dolu",
				"waited_ms", waited.Milliseconds(), "max_conns", st.MaxConns(), "acquired", st.AcquiredConns())
			return nil, ErrPoolExhausted
		}
		return nil, err
	}
	if waited > slowAcquire {
		d.logger.Warn("veritabanı bağlantısı geç alındı", "waited_ms", waited.Milliseconds(), "max_conns", d.pool.Stat().MaxConns())
	}
	return conn, nil
}

// Read, tek bir bağlantıyla transaction'sız okuma/yazma yapar.
func (d *DB) Read(ctx context.Context, fn func(q Querier) error) error {
	conn, err := d.acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	return fn(conn)
}

// Tx, fn'yi tek bir transaction içinde çalıştırır (READ COMMITTED; satır
// kilitleri "for update" ile alınır, ADR-0004). fn hata dönerse veya panic
// olursa geri alınır; yoksa commit edilir. Transaction kısa tutulmalıdır:
// ağ beklemesi veya yayın fn içinde yapılmaz (olaylar commit'ten sonra).
func (d *DB) Tx(ctx context.Context, fn func(tx pgx.Tx) error) (err error) {
	conn, err := d.acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
