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
func (e Env) Role() string   { return "bunker_" + string(e) + "_app" }

// ParseEnv, test veya prod kabul eder.
func ParseEnv(s string) (Env, error) {
	switch Env(s) {
	case EnvTest, EnvProd:
		return Env(s), nil
	}
	return "", fmt.Errorf("ortam test veya prod olmalı: %q", s)
}

// Rol düzeyi sınırları (ADR-0010, architecture.md §11).
const (
	StatementTimeout       = "5s"
	LockTimeout            = "3s"
	IdleInTxSessionTimeout = "10s"
	ConnectionLimit        = 10
)

var verifierRe = regexp.MustCompile(`^SCRAM-SHA-256\$[0-9]+:[A-Za-z0-9+/=]+\$[A-Za-z0-9+/=]+:[A-Za-z0-9+/=]+$`)

// SetupSQL, ortamın şemasını ve uygulama rolünü kuran, tekrar
// çalıştırılabilir SQL'i üretir (Supabase SQL editöründe postgres rolüyle).
// verifier, ScramVerifier çıktısıdır; düz şifre değildir.
func SetupSQL(env Env, verifier string) (string, error) {
	return setupSQL(env.Schema(), env.Role(), verifier)
}

var identRe = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)

func setupSQL(schema, role, verifier string) (string, error) {
	if !verifierRe.MatchString(verifier) {
		return "", fmt.Errorf("geçersiz SCRAM doğrulayıcısı")
	}
	if !identRe.MatchString(schema) || !identRe.MatchString(role) {
		return "", fmt.Errorf("geçersiz şema veya rol adı")
	}
	r := strings.NewReplacer(
		"{{schema}}", schema,
		"{{role}}", role,
		"{{verifier}}", verifier,
		"{{stmt}}", StatementTimeout,
		"{{lock}}", LockTimeout,
		"{{idle}}", IdleInTxSessionTimeout,
		"{{connlimit}}", fmt.Sprint(ConnectionLimit),
	)
	return r.Replace(setupTemplate), nil
}

// setupTemplate: şifre yalnızca SCRAM doğrulayıcısı olarak verilir.
// Uygulama rolü şemada nesne oluşturamaz (CREATE yok); tabloları migration
// rolü kurar (DB-002).
const setupTemplate = `-- Sığınak: {{schema}} şeması ve {{role}} rolü (DB-001b).
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

grant usage on schema {{schema}} to {{role}};

commit;

-- Doğrulama: tek satır; rolcanlogin=true, diğer tüm nitelikler false olmalı.
select rolname, rolcanlogin, rolsuper, rolbypassrls, rolcreaterole, rolcreatedb,
       rolreplication, rolconnlimit, rolconfig
  from pg_roles where rolname = '{{role}}';
`
