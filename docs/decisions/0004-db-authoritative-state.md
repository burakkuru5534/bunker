# ADR-0004: Veritabanı otoriteli durum — satır kilidi, sürüm, idempotency

- **Durum:** Kabul edildi (2026-11-02)

## Bağlam
Oyuncular aynı anda oy verebilir, kart açabilir; zamanlayıcı da aynı anda aşamayı ilerletmek isteyebilir. Ağ tekrarları aynı isteği birden fazla gönderebilir. Sunucu yeniden başlayabilir.

## Karar
- Her oyun eylemi kısa bir transaction içinde işlenir: `games` satırı `SELECT … FOR UPDATE` ile kilitlenir → engine çağrılır → değişiklikler yazılır → `state_version` +1 → commit.
- Bellekte oda başına mutex, veritabanında kilit beklemesini azaltır (doğruluk için gerekli değil; kilit veritabanındadır).
- İstemci her eylemde `action_id` (UUID) gönderir; `(game_id, player_id, action_id)` tekildir. Tekrarlanan istek önceki sonucu döndürür.
- Oy için ayrıca `(game_id, round, voter_id)` tekil kısıtı.
- Olaylar yalnızca commit başarılı olduktan sonra yayınlanır.
- Uzun tartışmalar veya WS bağlantıları boyunca transaction açık tutulmaz.

## Alternatifler
- **Yalnızca bellek içi aktör (oda başına goroutine):** Hızlı ama yeniden başlatmada durum kaybolur; yine de kalıcılık gerekir.
- **Optimistic concurrency (yalnızca sürüm kontrolü, kilitsiz):** Yüksek çakışmada çok sayıda yeniden deneme; oylama anında tüm oyuncular aynı anda yazdığı için uygun değil.

## Sonuçlar
- Doğruluk veritabanı tarafından garanti edilir; sunucu yeniden başlasa da durum kaybolmaz.
- Maç başına yazma sayısı ölçülmeli (QA-007, maliyet).
