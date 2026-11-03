// Command migrate, migrations/postgres altındaki SQL migration'larını bir
// ortamın şemasına uygular (DB-002). Ayrıntı: migrations/postgres/README.md.
//
//	go run ./cmd/migrate status            # uygulanmış / bekleyen
//	go run ./cmd/migrate up -dry-run       # ne uygulanacağını göster
//	go run ./cmd/migrate up                # uygula + yetkileri eşitle
//	go run ./cmd/migrate new create_users  # sıradaki dosyayı oluştur
//
// Bağlantı: MIGRATION_DATABASE_URL (.env; prod için .env.prod). Yalnızca
// bunker_<ortam>_migrator rolüyle, verify-full TLS ile çalışır.
// Prod'a uygulamak için -yes-prod gerekir (önce test'e uygulanır, ADR-0009).
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/config"
	"github.com/burakkuru5534/bunker/backend/internal/dbsetup"
	"github.com/burakkuru5534/bunker/backend/internal/migrate"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
)

const defaultDir = "../migrations/postgres"

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "hata:", err)
		os.Exit(1)
	}
}

const usage = `kullanım:
  migrate status [-env test] [-dir ../migrations/postgres]
  migrate up     [-env test] [-dry-run] [-yes-prod]
  migrate new    <ad>        [-dir ../migrations/postgres]
  migrate lint               [-dir ../migrations/postgres]`

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	envName := fs.String("env", "test", "ortam: test | prod")
	dir := fs.String("dir", defaultDir, "migration klasörü")
	envFile := fs.String("env-file", "", "bağlantı dosyası (varsayılan: test → .env, prod → .env.prod)")
	dryRun := fs.Bool("dry-run", false, "up: yalnızca bekleyenleri göster")
	yesProd := fs.Bool("yes-prod", false, "up: prod'a uygulamayı onayla")
	strict := fs.Bool("strict", true, "verify-full TLS ve rol denetimleri (yalnızca yerel testte false)")

	cmd := args[0]
	rest := args[1:]
	var name string
	if cmd == "new" {
		if len(rest) == 0 || rest[0] == "" || rest[0][0] == '-' {
			return errors.New("migrate new <ad>: ad gerekli (ör. create_users)")
		}
		name, rest = rest[0], rest[1:]
	}
	if err := fs.Parse(rest); err != nil {
		return err
	}

	switch cmd {
	case "new":
		return runNew(*dir, name, stdout)
	case "lint":
		migs, err := migrate.Load(*dir)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "✓ %d migration kurallara uygun\n", len(migs))
		return nil
	case "status", "up":
	default:
		return fmt.Errorf("bilinmeyen komut %q\n%s", cmd, usage)
	}

	env, err := dbsetup.ParseEnv(*envName)
	if err != nil {
		return err
	}
	if cmd == "up" && env == dbsetup.EnvProd && !*dryRun && !*yesProd {
		return errors.New("prod'a uygulamak için -yes-prod gerekli (önce test'e uygulayın ve doğrulayın)")
	}
	migs, err := migrate.Load(*dir)
	if err != nil {
		return err
	}

	spec := env.Spec(dbsetup.KindMigrator)
	if *envFile == "" {
		*envFile = ".env"
		if env == dbsetup.EnvProd {
			*envFile = ".env.prod"
		}
	}
	values, err := config.ReadDotEnv(*envFile)
	if err != nil {
		return err
	}
	dbURL := config.Secret(config.WithFallback(getenv, values)(spec.EnvKey))
	if !dbURL.IsSet() {
		return fmt.Errorf("%s bulunamadı (%s veya ortam değişkeni); önce: go run ./cmd/dbsetup init -role migrator", spec.EnvKey, *envFile)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	pool, err := persistence.Open(ctx, persistence.Options{
		URL: dbURL.Reveal(), MaxConns: 1, Schema: env.Schema(), Strict: *strict, AppName: "siginak-migrate",
	}, logger)
	if err != nil {
		return err
	}
	defer pool.Close()

	var user string
	if err := pool.QueryRow(ctx, "select current_user::text").Scan(&user); err != nil {
		return err
	}
	if user != spec.Name {
		return fmt.Errorf("bağlanılan rol %q, beklenen %q (%s yanlış rolü gösteriyor)", user, spec.Name, spec.EnvKey)
	}

	opts := migrate.Options{Schema: env.Schema(), AppRole: env.Spec(dbsetup.KindApp).Name, Logger: logger}
	fmt.Fprintf(stdout, "ortam %s, şema %s, %d migration dosyası\n", env, env.Schema(), len(migs))

	if cmd == "status" {
		st, err := migrate.GetStatus(ctx, pool, opts, migs)
		if err != nil {
			return err
		}
		for _, a := range st.Applied {
			fmt.Fprintf(stdout, "✓ %04d_%s  %s  (%d ms)\n", a.Version, a.Name, a.AppliedAt.UTC().Format(time.RFC3339), a.DurationMS)
		}
		for _, m := range st.Pending {
			fmt.Fprintf(stdout, "· %s  bekliyor\n", m.File)
		}
		fmt.Fprintf(stdout, "%d uygulanmış, %d bekliyor\n", len(st.Applied), len(st.Pending))
		return nil
	}

	applied, rep, err := migrate.Up(ctx, pool, opts, migs, *dryRun)
	if *dryRun {
		if err != nil {
			return err
		}
		for _, m := range applied {
			fmt.Fprintf(stdout, "· %s uygulanacak\n", m.File)
		}
		fmt.Fprintf(stdout, "%d migration uygulanacak (dry-run: değişiklik yapılmadı)\n", len(applied))
		return nil
	}
	for _, m := range applied {
		fmt.Fprintf(stdout, "✓ %s\n", m.File)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "%d migration uygulandı; yetkiler eşitlendi: %d tablo, RLS açılan %d, politika eklenen %d\n",
		len(applied), rep.Tables, len(rep.RLSEnabled), len(rep.PoliciesCreated))
	return nil
}

func runNew(dir, name string, stdout io.Writer) error {
	file, body, err := migrate.NextFile(dir, name)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, file)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(body); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	fmt.Fprintln(stdout, path)
	return nil
}
