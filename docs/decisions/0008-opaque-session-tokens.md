# ADR-0008: Opak, sunucu tarafında doğrulanan misafir oturum token'ları

- **Durum:** Kabul edildi (2026-11-02)

## Bağlam
MVP'de misafir oturumu gerekiyor; Firebase Auth yasak. Token'lar süreli, iptal edilebilir ve sunucu tarafında doğrulanabilir olmalı. Tek instance ve PostgreSQL var.

## Karar
- 32 bayt `crypto/rand` → base64url opak token.
- Veritabanında yalnızca `SHA-256(token)`; 90 gün geçerlilik; refresh ile döndürme; `revoked_at` ile iptal.
- Her istekte sunucu tarafında doğrulama, ≤60 sn bellek içi önbellek.

## Alternatifler
- **JWT (imzalı, durumsuz):** Veritabanı sorgusu gerektirmez ama iptal için yine de kara liste gerekir; imza anahtarı yönetimi ek secret ve rotasyon yükü getirir. Tek instance + PostgreSQL ile kazancı yok.
- **Kısa ömürlü access + uzun ömürlü refresh token çifti:** Daha karmaşık; misafir hesap için gereksiz.

## Sonuçlar
- İptal anında (önbellekle en geç 60 sn) etkili.
- İstek başına bir hafif sorgu (önbellekle çoğu zaman sıfır).
- Token'lar yüksek entropili olduğu için hash'te tuz/pepper gerekmez; ek secret yok.
