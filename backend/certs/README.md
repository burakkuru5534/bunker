# Sertifikalar

| Dosya | Kaynak | Gizli mi? |
|---|---|---|
| `supabase-ca.crt` | Supabase panelinde **Database Settings → SSL Configuration → Download certificate** (`prod-ca-2021.crt`), bu adla kaydedilir | Hayır — herkese açık kök CA sertifikasıdır, git'e girer |

`DATABASE_URL` içindeki `sslrootcert=certs/supabase-ca.crt` bu dosyayı gösterir (yol `backend/` klasörüne göredir). Sunucu `sslmode=verify-full` ile Supabase'in sertifikasını ve sunucu adını bu CA ile doğrular; test/prod'da başka bir mod kabul edilmez.

Buraya **asla** özel anahtar (`*.key`, `*.pem`) konmaz; `.gitignore` bunları dışlar.
