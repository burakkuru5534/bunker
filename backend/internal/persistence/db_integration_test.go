package persistence

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// testDB, yönetici adresiyle (yerel mod) maxConns'lı bir DB açar ve geçici
// bir tablo kurar.
func testDB(t *testing.T, maxConns int32, acquire time.Duration) (*DB, *bytes.Buffer, string) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_ADMIN_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_ADMIN_URL tanımlı değil")
	}
	var logs bytes.Buffer
	logger := quietLogger(&logs)
	pool, err := Open(ctx5(t), Options{URL: url, MaxConns: maxConns, Schema: "public"}, logger)
	if err != nil {
		t.Fatal(err)
	}
	db := NewDB(pool, acquire, logger)
	t.Cleanup(db.Close)

	table := "it_db_" + strings.ReplaceAll(t.Name(), "/", "_")
	table = strings.ToLower(table)
	if err := db.Read(ctx5(t), func(q Querier) error {
		_, err := q.Exec(context.Background(), "drop table if exists "+table+"; create table "+table+" (id int primary key)")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = NewDB(pool, time.Second, logger).Read(context.Background(), func(q Querier) error {
			_, err := q.Exec(context.Background(), "drop table if exists "+table)
			return err
		})
	})
	return db, &logs, table
}

func count(t *testing.T, db *DB, table string) int {
	t.Helper()
	var n int
	if err := db.Read(ctx5(t), func(q Querier) error {
		return q.QueryRow(context.Background(), "select count(*) from "+table).Scan(&n)
	}); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestIntegrationTxCommitRollbackPanic(t *testing.T) {
	db, _, table := testDB(t, 2, time.Second)
	ctx := ctx5(t)
	ins := func(tx pgx.Tx, id int) error {
		_, err := tx.Exec(ctx, "insert into "+table+" values ($1)", id)
		return err
	}

	if err := db.Tx(ctx, func(tx pgx.Tx) error { return ins(tx, 1) }); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("iş kuralı")
	if err := db.Tx(ctx, func(tx pgx.Tx) error {
		if err := ins(tx, 2); err != nil {
			return err
		}
		return boom
	}); !errors.Is(err, boom) {
		t.Fatalf("fn hatası dönmeli: %v", err)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("panic yeniden fırlatılmalı")
			}
		}()
		_ = db.Tx(ctx, func(tx pgx.Tx) error {
			_ = ins(tx, 3)
			panic("beklenmeyen")
		})
	}()
	// Kısıt ihlali: hata döner, transaction geri alınır.
	if err := db.Tx(ctx, func(tx pgx.Tx) error {
		_ = ins(tx, 4)
		return ins(tx, 1)
	}); err == nil {
		t.Fatal("tekil ihlal hata vermeli")
	}
	if n := count(t, db, table); n != 1 {
		t.Fatalf("yalnızca ilk transaction kalmalı: %d", n)
	}
	// Bağlantılar havuza döndü (panic ve hatadan sonra da).
	if st := db.Stats(); st.AcquiredConns() != 0 {
		t.Fatalf("bağlantı sızdı: %d", st.AcquiredConns())
	}
	if err := db.Ping(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationPoolExhaustedFailsFast(t *testing.T) {
	db, logs, _ := testDB(t, 1, 200*time.Millisecond)

	// Tek bağlantıyı tutan uzun iş.
	hold, release := make(chan struct{}), make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = db.Read(context.Background(), func(Querier) error {
			close(hold)
			<-release
			return nil
		})
	}()
	<-hold

	// İsteğin 10 sn'si olsa da 200 ms sonra hızlıca hata.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := time.Now()
	err := db.Tx(ctx, func(pgx.Tx) error { t.Fatal("çalışmamalı"); return nil })
	waited := time.Since(start)
	if !errors.Is(err, ErrPoolExhausted) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ErrPoolExhausted bekleniyordu: %v", err)
	}
	if waited < 150*time.Millisecond || waited > 2*time.Second {
		t.Fatalf("bekleme acquire süresine yakın olmalı: %v", waited)
	}
	if !strings.Contains(logs.String(), "veritabanı havuzu dolu") {
		t.Fatalf("uyarı loglanmalı: %s", logs.String())
	}

	// İstek iptal edildiyse havuz hatası değil, bağlam hatası döner.
	cctx, ccancel := context.WithCancel(context.Background())
	ccancel()
	if err := db.Read(cctx, func(Querier) error { return nil }); errors.Is(err, ErrPoolExhausted) || !errors.Is(err, context.Canceled) {
		t.Fatalf("iptal edilen istek: %v", err)
	}

	// Bağlantı geç bırakılırsa bekleyen alır ve "geç alındı" uyarısı yazılır.
	slow := NewDB(db.Pool(), 3*time.Second, db.logger)
	go func() { time.Sleep(700 * time.Millisecond); close(release) }()
	if err := slow.Read(context.Background(), func(Querier) error { return nil }); err != nil {
		t.Fatalf("bağlantı bırakılınca alınmalı: %v", err)
	}
	wg.Wait()
	if !strings.Contains(logs.String(), "geç alındı") {
		t.Fatalf("geç alma uyarısı: %s", logs.String())
	}
}

func TestNewDBDefaults(t *testing.T) {
	d := NewDB(nil, 0, nil)
	if d.acquireTimeout != DefaultAcquireTimeout || d.logger == nil {
		t.Fatal("varsayılanlar")
	}
}
