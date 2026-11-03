package dbsetup

import (
	"fmt"
	"regexp"
	"strings"
)

// Env, kurulacak ortamdır.
type Env string

const (
	EnvTest Env = "test"
	EnvProd Env = "prod"
)

// Schema ve Role, ortamın şema ve uygulama rolü adlarıdır (ADR-0009).
func (e Env) Schema() string { return "bunker_" + string(e) }
func (e Env) Role() string   { return e.Spec(KindApp).Name }

// ParseEnv, test veya prod kabul eder.
func ParseEnv(s string) (Env, error) {
	switch Env(s) {
	case EnvTest, EnvProd:
		return Env(s), nil
	}
	return "", fmt.Errorf("ortam test veya prod olmalı: %q", s)
}

// Kind, rol türüdür.
type Kind string

const (
	// KindApp: sunucunun kullandığı rol. Yalnızca veri okur/yazar (DML).
	KindApp Kind = "app"
	// KindMigrator: migration rolü. Şemada tablo oluşturur ve sahibidir (DDL);
	// uygulama rolüne yetkileri cmd/migrate verir (DB-002).
	KindMigrator Kind = "migrator"
)

// ParseKind, app veya migrator kabul eder.
func ParseKind(s string) (Kind, error) {
	switch Kind(s) {
	case KindApp, KindMigrator:
		return Kind(s), nil
	}
	return "", fmt.Errorf("rol türü app veya migrator olmalı: %q", s)
}

// RoleSpec, bir rolün adı, bağlantı adresinin yazıldığı anahtar ve rol düzeyi
// sınırlarıdır (ADR-0010, architecture.md §11).
type RoleSpec struct {
	Kind             Kind
	Name             string
	EnvKey           string // .env anahtarı
	StatementTimeout string
	LockTimeout      string
	IdleInTxTimeout  string
	ConnectionLimit  int
	SchemaPrivileges string // "usage" | "usage, create"
}

// CanCreate, rol şemada nesne oluşturabilir mi?
func (r RoleSpec) CanCreate() bool { return r.Kind == KindMigrator }

// Uygulama rolünün sınırları.
const (
	StatementTimeout       = "5s"
	LockTimeout            = "3s"
	IdleInTxSessionTimeout = "10s"
	ConnectionLimit        = 10
)

// Spec, ortamın verilen türdeki rolünü döndürür.
func (e Env) Spec(k Kind) RoleSpec {
	if k == KindMigrator {
		return RoleSpec{
			Kind: k, Name: "bunker_" + string(e) + "_migrator", EnvKey: "MIGRATION_DATABASE_URL",
			// Migration'lar daha uzun sürebilir (1min = 60 sn; PostgreSQL bu biçimde
			// gösterir ve check birebir karşılaştırır). Kilit beklemesi kısa tutulur
			// ki canlı trafiği uzun süre bloklamasın.
			StatementTimeout: "1min", LockTimeout: "5s", IdleInTxTimeout: "1min",
			ConnectionLimit: 2, SchemaPrivileges: "usage, create",
		}
	}
	return RoleSpec{
		Kind: KindApp, Name: "bunker_" + string(e) + "_app", EnvKey: "DATABASE_URL",
		StatementTimeout: StatementTimeout, LockTimeout: LockTimeout, IdleInTxTimeout: IdleInTxSessionTimeout,
		ConnectionLimit: ConnectionLimit, SchemaPrivileges: "usage",
	}
}

var verifierRe = regexp.MustCompile(`^SCRAM-SHA-256\$[0-9]+:[A-Za-z0-9+/=]+\$[A-Za-z0-9+/=]+:[A-Za-z0-9+/=]+$`)

// SetupSQL, ortamın şemasını ve verilen rolü kuran, tekrar çalıştırılabilir
// SQL'i üretir (Supabase SQL editöründe postgres rolüyle). verifier,
// ScramVerifier çıktısıdır; düz şifre değildir.
func SetupSQL(env Env, kind Kind, verifier string) (string, error) {
	return setupSQL(env.Schema(), env.Spec(kind), verifier)
}

var identRe = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

func setupSQL(schema string, spec RoleSpec, verifier string) (string, error) {
	if !verifierRe.MatchString(verifier) {
		return "", fmt.Errorf("geçersiz SCRAM doğrulayıcısı")
	}
	if !identRe.MatchString(schema) || !identRe.MatchString(spec.Name) {
		return "", fmt.Errorf("geçersiz şema veya rol adı")
	}
	r := strings.NewReplacer(
		"{{schema}}", schema,
		"{{role}}", spec.Name,
		"{{verifier}}", verifier,
		"{{stmt}}", spec.StatementTimeout,
		"{{lock}}", spec.LockTimeout,
		"{{idle}}", spec.IdleInTxTimeout,
		"{{connlimit}}", fmt.Sprint(spec.ConnectionLimit),
		"{{privs}}", spec.SchemaPrivileges,
	)
	return r.Replace(setupTemplate), nil
}

// setupTemplate: şifre yalnızca SCRAM doğrulayıcısı olarak verilir.
// Uygulama rolü şemada nesne oluşturamaz (yalnızca USAGE); tabloları migration
// rolü (USAGE, CREATE) kurar (DB-002).
const setupTemplate = `-- Sığınak: {{schema}} şeması ve {{role}} rolü (DB-001b, DB-002).
-- Supabase > SQL Editor'de çalıştırın. Tekrar çalıştırmak güvenlidir (şifreyi yeniler).
-- Bu metin düz şifre içermez; yine de snippet olarak KAYDETMEYİN.
begin;

do $$
begin
  if not exists (select 1 from pg_roles where rolname = '{{role}}') then
    create role {{role}} nologin;
  end if;
end
$$;

-- SUPERUSER / BYPASSRLS / CREATEROLE vb. belirtilmez: yeni roller bunlarda
-- güvenli varsayılanla (hayır) oluşur; Supabase'in postgres rolü (süper
-- kullanıcı değil) bu nitelikleri değiştiremez. Sonuç en alttaki sorguyla ve
-- sunucunun açılış denetimiyle doğrulanır.
alter role {{role}} with login connection limit {{connlimit}}
  password '{{verifier}}';

alter role {{role}} set search_path = {{schema}};
alter role {{role}} set statement_timeout = '{{stmt}}';
alter role {{role}} set lock_timeout = '{{lock}}';
alter role {{role}} set idle_in_transaction_session_timeout = '{{idle}}';

create schema if not exists {{schema}};
revoke all on schema {{schema}} from public;

-- Supabase Data API rolleri bu şemaya erişemez.
do $$
declare r text;
begin
  foreach r in array array['anon', 'authenticated', 'service_role'] loop
    if exists (select 1 from pg_roles where rolname = r) then
      execute format('revoke all on schema {{schema}} from %I', r);
    end if;
  end loop;
end
$$;

grant {{privs}} on schema {{schema}} to {{role}};

commit;

-- Doğrulama: tek satır; rolcanlogin=true, diğer tüm nitelikler false olmalı.
select rolname, rolcanlogin, rolsuper, rolbypassrls, rolcreaterole, rolcreatedb,
       rolreplication, rolconnlimit, rolconfig
  from pg_roles where rolname = '{{role}}';
`
