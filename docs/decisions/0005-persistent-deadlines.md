# ADR-0005: Kalıcı deadline'larla zamanlayıcı kurtarma

- **Durum:** Kabul edildi (2026-11-02)

## Bağlam
Her aşamanın süresi var (kart açma 20 sn, tartışma 90/120 sn vb.). Zamanlayıcılar yalnızca bellekte olursa sunucu yeniden başladığında oyunlar takılır.

## Karar
- Aşama bitiş zamanı `games.phase_deadline_at` alanında saklanır.
- Bellekte yalnızca aktif oyunları izleyen hafif bir zamanlayıcı çalışır; süre dolunca sistem adına "aşama süresi doldu" eylemi, beklenen `state_version` ile normal eylem yolundan işlenir. Sürüm uyuşmazsa (aşama zaten ilerlemiş) eylem yok sayılır.
- Sunucu açılışında aktif oyunlar yüklenir, süresi geçmiş aşamalar hemen işlenir, diğerleri yeniden planlanır.
- Ek güvence: bir oyuncu eylem gönderdiğinde veya yeniden bağlandığında deadline kontrolü yapılır.

## Alternatifler
- **Yalnızca bellek içi timer:** Yeniden başlatmada kayıp.
- **Harici zamanlayıcı (Cloud Scheduler / Cloud Tasks):** Saniye hassasiyeti için uygun değil, ek maliyet ve karmaşıklık.
- **Veritabanı polling'i:** Gereksiz sorgu yükü.

## Sonuçlar
- Zamanlayıcı ile oyuncu eylemi aynı anda gelirse ADR-0004'teki kilit ve sürüm kontrolü tutarlılığı sağlar.
- Cloud Run'da instance sıfıra inerse zamanlayıcı çalışmaz; ilk istek/yeniden bağlanmada ve açılışta deadline'lar işlenir. Cloud Run CPU tahsis modu ARCH-005'te belirlenir.
