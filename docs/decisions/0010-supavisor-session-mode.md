# ADR-0010: Supabase'e Supavisor session mode ile bağlanma

- **Durum:** Kabul edildi (2026-11-02)

## Bağlam
Go backend Cloud Run'da uzun ömürlü bir süreç olarak çalışır ve pgx ile prepared statement kullanır. Supabase'in doğrudan bağlantısı Free/Pro planlarda yalnızca IPv6'dır; Cloud Run'ın varsayılan internet çıkışı IPv4'tür. IPv4 eklentisi ücretlidir.

## Karar
- Paylaşımlı Supavisor pooler, **session mode, port 5432** (IPv4).
- Kullanıcı adı `<rol>.<proje-ref>`; ortam başına ayrı rol (ADR-0009).
- pgxpool: prod `MaxConns=5`, test `MaxConns=3`, `MinConns=0`.
- Rol düzeyinde `statement_timeout=5s`, `idle_in_transaction_session_timeout=10s`, `lock_timeout=3s`.
- TLS zorunlu; hedef `verify-full`.

## Alternatifler
- **Doğrudan bağlantı + IPv4 eklentisi:** Ek ücret; ölçek gerekmedikçe gereksiz.
- **Doğrudan bağlantı + Cloud Run Direct VPC egress (dual-stack IPv6):** VPC/subnet kurulumu ve ek yapılandırma; MVP için karmaşık.
- **Transaction mode (6543):** Prepared statement, session state desteklemez; serverless için tasarlanmış.
- **Dedicated pooler:** Yalnızca Pro ve üzeri.

## Sonuçlar
- Ek maliyet yok, IPv4 ile çalışır, prepared statement'lar kullanılabilir.
- Session mode'da her havuz bağlantısı bir arka uç bağlantısı tutar → `MaxConns` küçük tutulur; pooler havuz boyutu DB-005'te panelden doğrulanır.
- Pooler ek bir ağ atlaması ekler (birkaç ms); oyun için önemsiz.
