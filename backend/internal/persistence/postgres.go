// Package persistence, PostgreSQL bağlantı havuzunu kurar ve doğrular
// (ADR-0009, ADR-0010, docs/architecture.md §11).
//
// Supabase'e Supavisor session mode (5432, IPv4) ile pgxpool üzerinden
// bağlanılır. Açılışta bağlantının güvenli olduğu doğrulanır: TLS düzeyi,
// rolün yetkisi ve beklenen şema. Bağlantı adresi hiçbir hata veya loga
// yazılmaz.
package persistence

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Havuz ayarları (architecture.md §11 "Bağlantı havuzu").
const (
	maxConnIdleTime   = 5 * time.Minute
	maxConnLifetime   = 30 * time.Minute
	healthCheckPeriod = time.Minute
	connectTimeout    = 5 * time.Second
)

// Options, havuz ayarlarıdır.
type Options struct {
	// URL, bağlantı adresi (gizli). Yalnızca sürücüye verilir.
	URL string
	// MaxConns, havuz üst sınırı (prod 5, test 3).
	MaxConns int32
	// Schema, rolün search_path'inde beklenen şema (ör. bunker_test).
	Schema string
	// Strict: test/prod. Doğrulanmış TLS (sslmode=verify-full) zorunlu ve
	// süper kullanıcı / RLS atlayan / Supabase yönetim rolleriyle bağlantı
	// reddedilir. Yerelde (false) yalnızca uyarı loglanır; yerel olmayan
	// sunucuya şifresiz bağlantı yine reddedilir.
	Strict bool
	// AppName, sunucu tarafında görünen application_name.
	AppName string
}

// Open, havuzu kurar ve Verify ile doğrular. ctx, ilk bağlantı ve doğrulama
// için üst süreyi belirler. Hata durumunda havuz kapatılır.
func Open(ctx context.Context, opts Options, logger *slog.Logger) (*pgxpool.Pool, error) {
	cfg, err := poolConfig(opts, logger)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("veritabanı havuzu kurulamadı: %w", err)
	}
	if err := Verify(ctx, pool, opts); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// poolConfig, adresi ayrıştırır, TLS ilkesini uygular ve havuz ayarlarını yazar.
func poolConfig(opts Options, logger *slog.Logger) (*pgxpool.Config, error) {
	if opts.URL == "" {
		return nil, errors.New("DATABASE_URL boş")
	}
	cfg, err := pgxpool.ParseConfig(opts.URL)
	if err != nil {
		// pgx/url hataları adresin parçalarını içerebilir; ayrıntı verilmez.
		return nil, errors.New("DATABASE_URL ayrıştırılamadı (biçimi kontrol edin: postgres://kullanıcı:şifre@sunucu:5432/postgres?sslmode=verify-full)")
	}
	if err := checkTLS(&cfg.ConnConfig.Config, opts.Strict, logger); err != nil {
		return nil, err
	}

	if opts.MaxConns < 1 {
		return nil, fmt.Errorf("MaxConns en az 1 olmalı: %d", opts.MaxConns)
	}
	cfg.MaxConns = opts.MaxConns
	cfg.MinConns = 0
	cfg.MaxConnIdleTime = maxConnIdleTime
	cfg.MaxConnLifetime = maxConnLifetime
	cfg.HealthCheckPeriod = healthCheckPeriod
	if cfg.ConnConfig.ConnectTimeout == 0 {
		cfg.ConnConfig.ConnectTimeout = connectTimeout
	}
	if opts.AppName != "" {
		cfg.ConnConfig.RuntimeParams["application_name"] = opts.AppName
	}
	return cfg, nil
}

// tlsLevel, bağlantının en zayıf TLS düzeyidir.
type tlsLevel int

const (
	tlsNone      tlsLevel = iota // şifresiz (sslmode=disable, ya da prefer/allow geri düşüşü)
	tlsEncrypted                 // şifreli ama sertifika/ad doğrulanmıyor (require, verify-ca)
	tlsVerified                  // sertifika ve sunucu adı doğrulanıyor (verify-full)
)

func (l tlsLevel) String() string {
	return [...]string{"şifresiz", "şifreli (doğrulamasız)", "doğrulanmış"}[l]
}

func levelOf(t *tls.Config) tlsLevel {
	switch {
	case t == nil:
		return tlsNone
	case t.InsecureSkipVerify:
		// require ve verify-ca: verify-ca zinciri doğrular ama adı doğrulamaz.
		return tlsEncrypted
	default:
		return tlsVerified
	}
}

// weakestTLS, birincil bağlantı ve tüm geri düşüşler arasındaki en zayıf düzeydir
// (sslmode=prefer, şifresiz geri düşüş içerir).
func weakestTLS(cc *pgconn.Config) tlsLevel {
	lvl := levelOf(cc.TLSConfig)
	for _, fb := range cc.Fallbacks {
		lvl = min(lvl, levelOf(fb.TLSConfig))
	}
	return lvl
}

func allLoopback(cc *pgconn.Config) bool {
	hosts := []string{cc.Host}
	for _, fb := range cc.Fallbacks {
		hosts = append(hosts, fb.Host)
	}
	for _, h := range hosts {
		if !isLoopback(h) {
			return false
		}
	}
	return true
}

func isLoopback(host string) bool {
	if strings.HasPrefix(host, "/") { // unix soketi
		return true
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func checkTLS(cc *pgconn.Config, strict bool, logger *slog.Logger) error {
	lvl := weakestTLS(cc)
	switch {
	case lvl == tlsVerified:
		return nil
	case strict:
		return fmt.Errorf("veritabanı TLS düzeyi %s; test/prod'da sslmode=verify-full (ve sslrootcert) zorunlu", lvl)
	case allLoopback(cc):
		return nil // yerel makinedeki veritabanı
	case lvl == tlsNone:
		return errors.New("yerel olmayan veritabanına şifresiz bağlantı reddedildi; sslmode=require veya verify-full kullanın")
	default:
		logger.Warn("veritabanı sertifikası doğrulanmıyor; sslmode=verify-full önerilir", "tls", lvl.String())
		return nil
	}
}

// managementRoles, uygulamanın asla kullanmaması gereken Supabase/PostgreSQL
// yönetim rolleridir (security.md §2).
var managementRoles = map[string]bool{
	"postgres": true, "supabase_admin": true, "service_role": true,
	"authenticator": true, "anon": true, "authenticated": true,
	"supabase_auth_admin": true, "supabase_storage_admin": true,
	"supabase_realtime_admin": true, "supabase_replication_admin": true,
	"dashboard_user": true, "pgbouncer": true,
}

// Identity, bağlantının kimlik bilgisidir.
type Identity struct {
	User      string
	Schema    string // current_schema(); search_path'te geçerli şema yoksa boş
	Super     bool
	BypassRLS bool
}

const identityQuery = `
select current_user::text,
       coalesce(current_schema()::text, ''),
       r.rolsuper,
       r.rolbypassrls
  from pg_catalog.pg_roles r
 where r.rolname = current_user`

// Verify, bir bağlantı alıp kimliği denetler:
//   - current_schema() beklenen şema olmalı (rolün search_path'i, ADR-0009)
//   - Strict ise rol süper kullanıcı, RLS atlayan veya yönetim rolü olmamalı
func Verify(ctx context.Context, pool *pgxpool.Pool, opts Options) error {
	var id Identity
	err := pool.QueryRow(ctx, identityQuery).Scan(&id.User, &id.Schema, &id.Super, &id.BypassRLS)
	if err != nil {
		return fmt.Errorf("veritabanına bağlanılamadı: %w", err)
	}
	return checkIdentity(id, opts)
}

func checkIdentity(id Identity, opts Options) error {
	if opts.Strict {
		switch {
		case id.Super:
			return fmt.Errorf("rol %q süper kullanıcı; uygulama kendi rolüyle bağlanmalı (bunker_<ortam>_app)", id.User)
		case id.BypassRLS:
			return fmt.Errorf("rol %q RLS'i atlıyor (BYPASSRLS); uygulama kendi rolüyle bağlanmalı", id.User)
		case managementRoles[id.User] || strings.HasPrefix(id.User, "supabase_"):
			return fmt.Errorf("rol %q bir yönetim rolü; uygulama kendi rolüyle bağlanmalı", id.User)
		}
	}
	if id.Schema != opts.Schema {
		got := id.Schema
		if got == "" {
			got = "(yok)"
		}
		return fmt.Errorf("beklenen şema %q, rolün geçerli şeması %s; rolün search_path'ini ve şemanın varlığını kontrol edin", opts.Schema, got)
	}
	return nil
}
