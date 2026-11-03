// Command dbsetup, bir ortamın Supabase şemasını ve uygulama rolünü kurmaya
// ve doğrulamaya yardım eder (DB-001b). Ayrıntı: backend/README.md.
//
//	go run ./cmd/dbsetup init  -env test [-role app|migrator] -ref <proje-ref> -host <pooler-host>
//	go run ./cmd/dbsetup check [-role app|migrator]
//
// Roller (ADR-0009, DB-002): app (sunucu; DATABASE_URL) ve migrator
// (migration; MIGRATION_DATABASE_URL). İkisi de aynı .env'e yazılır.
//
// init: güçlü bir şifre üretir, DATABASE_URL'i .env'e yazar (şifre ekrana
// basılmaz) ve Supabase SQL editöründe çalıştırılacak kurulum SQL'ini stdout'a
// yazar. SQL'de şifre yalnızca SCRAM-SHA-256 doğrulayıcısı olarak bulunur.
// Bilgi mesajları stderr'e gider; böylece çıktı doğrudan panoya alınabilir:
//
//	go run ./cmd/dbsetup init ... | Set-Clipboard     (PowerShell)
//
// check: .env'deki adresle katı denetimlerle bağlanır (verify-full TLS, rol,
// şema) ve rol ayarlarını/yetkilerini raporlar.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/config"
	"github.com/burakkuru5534/bunker/backend/internal/dbsetup"
	"github.com/burakkuru5534/bunker/backend/internal/persistence"
)

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "hata:", err)
		os.Exit(1)
	}
}

const usage = `kullanım:
  dbsetup init  -env test [-role app|migrator] -ref <proje-ref> -host <pooler-host> [-cert certs/supabase-ca.crt] [-force]
  dbsetup check [-env test] [-role app|migrator] [-env-file .env] [-strict=true]`

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New(usage)
	}
	switch args[0] {
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], getenv, stdout, stderr)
	default:
		return fmt.Errorf("bilinmeyen komut %q\n%s", args[0], usage)
	}
}

func runInit(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	envName := fs.String("env", "test", "ortam: test | prod")
	roleName := fs.String("role", "app", "rol: app | migrator")
	ref := fs.String("ref", "", "Supabase proje kimliği (20 küçük harf)")
	host := fs.String("host", "", "Session pooler sunucusu (ör. aws-0-eu-central-1.pooler.supabase.com)")
	port := fs.Int("port", 5432, "port (session mode: 5432)")
	cert := fs.String("cert", "certs/supabase-ca.crt", "Supabase CA sertifikası (verify-full)")
	sslmode := fs.String("sslmode", "verify-full", "verify-full | require | disable (yalnızca yerel test)")
	envFile := fs.String("env-file", "", "adresin yazılacağı dosya (varsayılan: test → .env, prod → .env.prod)")
	force := fs.Bool("force", false, "dolu DATABASE_URL'in üzerine yaz (şifre yenileme)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	env, err := dbsetup.ParseEnv(*envName)
	if err != nil {
		return err
	}
	kind, err := dbsetup.ParseKind(*roleName)
	if err != nil {
		return err
	}
	spec := env.Spec(kind)
	*envFile = defaultEnvFile(*envFile, env)

	password, err := dbsetup.NewPassword(32)
	if err != nil {
		return err
	}
	p := dbsetup.ConnParams{Role: spec.Name, Password: password, ProjectRef: *ref, Host: *host,
		Port: *port, SSLMode: *sslmode, CertFile: *cert}
	if err := p.Validate(); err != nil {
		return err
	}
	if p.SSLMode == "verify-full" {
		info, err := dbsetup.InspectCert(p.CertFile)
		if err != nil {
			return err
		}
		if !info.IsCA {
			return fmt.Errorf("%s bir CA sertifikası değil (%s)", p.CertFile, info.Subject)
		}
		fmt.Fprintf(stderr, "CA sertifikası: %s (son geçerlilik %s)\n  SHA-256: %s\n", info.Subject, info.NotAfter, info.SHA256)
	}

	verifier, err := dbsetup.NewScramVerifier(password)
	if err != nil {
		return err
	}
	sql, err := dbsetup.SetupSQL(env, kind, verifier)
	if err != nil {
		return err
	}
	if err := dbsetup.UpsertEnvVar(*envFile, spec.EnvKey, p.URL(), *force); err != nil {
		return fmt.Errorf("%s: %w", spec.EnvKey, err)
	}
	fmt.Fprintf(stderr, "%s %s dosyasına yazıldı (şifre gösterilmez).\n", spec.EnvKey, *envFile)
	fmt.Fprintln(stderr, "Aşağıdaki SQL'i Supabase > SQL Editor'de çalıştırın; snippet olarak kaydetmeyin.")
	fmt.Fprintf(stderr, "Sonra: go run ./cmd/dbsetup check -role %s\n", kind)
	_, err = io.WriteString(stdout, sql)
	return err
}

func runCheck(args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	envName := fs.String("env", "test", "ortam: test | prod")
	roleName := fs.String("role", "app", "rol: app | migrator")
	envFile := fs.String("env-file", "", "adresin okunacağı dosya (varsayılan: test → .env, prod → .env.prod)")
	strict := fs.Bool("strict", true, "verify-full TLS ve rol denetimleri (yalnızca yerel testte false)")
	poolSize := fs.Int("pool-size", 0, "Supabase panelindeki Supavisor Pool Size (Database Settings); verilirse rolün ihtiyacıyla karşılaştırılır")
	if err := fs.Parse(args); err != nil {
		return err
	}
	env, err := dbsetup.ParseEnv(*envName)
	if err != nil {
		return err
	}
	kind, err := dbsetup.ParseKind(*roleName)
	if err != nil {
		return err
	}
	spec := env.Spec(kind)
	*envFile = defaultEnvFile(*envFile, env)
	values, err := config.ReadDotEnv(*envFile)
	if err != nil {
		return err
	}
	dbURL := config.Secret(config.WithFallback(getenv, values)(spec.EnvKey))
	if !dbURL.IsSet() {
		return fmt.Errorf("%s bulunamadı (%s veya ortam değişkeni)", spec.EnvKey, *envFile)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	logger := slog.New(slog.NewTextHandler(stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	pool, err := persistence.Open(ctx, persistence.Options{
		URL: dbURL.Reveal(), MaxConns: 1, Schema: env.Schema(), Strict: *strict, AppName: "siginak-dbsetup",
	}, logger)
	if err != nil {
		fmt.Fprintln(stdout, "✗ bağlantı:", err)
		return errors.New("denetim başarısız")
	}
	defer pool.Close()
	tls := "doğrulanmış (verify-full)"
	if !*strict {
		tls = "denetlenmedi (-strict=false)"
	}
	fmt.Fprintf(stdout, "✓ bağlantı: TLS %s, rol ve şema doğrulandı\n", tls)

	var user string
	if err := pool.QueryRow(ctx, "select current_user::text").Scan(&user); err != nil || user != spec.Name {
		fmt.Fprintf(stdout, "✗ rol: bağlanılan rol %q, beklenen %q (%s yanlış rolü gösteriyor)\n", user, spec.Name, spec.EnvKey)
		return errors.New("denetim başarısız")
	}
	results := dbsetup.Check(ctx, pool, env.Schema(), spec)
	if *poolSize > 0 {
		results = append(results, dbsetup.CheckPoolSize(*poolSize, spec))
	}
	for _, r := range results {
		fmt.Fprintf(stdout, "%s %s: %s\n", r.Status.Symbol(), r.Name, r.Detail)
	}
	if dbsetup.Failed(results) {
		return errors.New("denetim başarısız")
	}
	fmt.Fprintln(stdout, "Tüm denetimler geçti.")
	return nil
}

// defaultEnvFile: test → .env, prod → .env.prod (ikisi de git'e girmez).
func defaultEnvFile(given string, env dbsetup.Env) string {
	if given != "" {
		return given
	}
	if env == dbsetup.EnvProd {
		return ".env.prod"
	}
	return ".env"
}
