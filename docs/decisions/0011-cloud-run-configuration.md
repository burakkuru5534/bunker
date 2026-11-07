# ADR-0011: Cloud Run yapılandırması — instance-based, min 0 / max 1, europe-west1

- **Durum:** Kabul edildi (2026-11-02); 2026-11-07'de uygulandı. Arada değerlendirilen Render Free ([ADR-0013](0013-render-free-zero-cost.md)) uygulanmadı.

## Bağlam
Backend WebSocket bağlantıları ve bellek içi zamanlayıcılar kullanır (ADR-0003, ADR-0005). Cloud Run'da açık WS bağlantısı olan instance aktif sayılır ve instance-based faturalanır. Maliyet düşük ve öngörülebilir olmalı.

## Karar
- Bölge `europe-west1` (Tier 1). Servisler: `bunker-api-test`, `bunker-api`.
- **Instance-based faturalama** (CPU her zaman tahsisli).
- min-instances **0**, max-instances **1**, 1 vCPU / 512 MiB, eşzamanlılık 1000, istek zaman aşımı 3600 sn, session affinity açık.
- Startup probe `/health`; SIGTERM'de WS'ler `1012` ile kapatılır.
- Ortam başına ayrı servis ve servis hesabı. `DATABASE_URL` ortam değişkeni olarak verilir (güncelleme 2026-11-05: Secret Manager kullanılmaz; sonuçları security.md §3).

## Alternatifler
- **Request-based faturalama:** İstek yokken CPU kısıtlanır; zamanlayıcı goroutine'leri güvenilir çalışmaz. WS açıkken zaten instance-based faturalanır, fiyat avantajı kalmaz.
- **min-instances=1:** Soğuk başlatma olmaz ama 7/24 ücret (≈ $44/ay, Tier 1). Beta sonrası gerekirse yeniden değerlendirilir.
- **europe-west3 (Frankfurt):** Supabase'e daha yakın ama Tier 2 fiyat.

## Sonuçlar
- Boşta ücret yok; aylık tavan tek instance'ın 7/24 maliyeti.
- Bağlantı yokken instance kapanabilir; aktif oyunların deadline'ları bir sonraki açılışta işlenir.
- Cloud Run (Belçika) ↔ Supabase (Frankfurt) arasında birkaç ms gecikme; oyun için önemsiz.

## Uygulama (2026-11-07, REL-003)
- Proje: `siginak-aa1ccd` (yalnızca bu uygulama için; başka üye yok). Faturalama hesabı diğer projelerle ortaktır → Cloud Run ücretsiz kotası da ortaktır (o projelerde şu an Cloud Run servisi yok).
- İmaj: `backend/Dockerfile` (çok aşamalı, distroless/static, root olmayan kullanıcı, ~23 MB). Cloud Build ile derlenir, Artifact Registry `europe-west1/bunker` deposuna gider. Temizlik politikası: son 3 sürüm tutulur, 7 günden eskiler silinir (0,5 GB ücretsiz kota). Cloud Build kaynak kovası 3 günde temizlenir.
- Servis `bunker-api-test` — https://bunker-api-test-918364240578.europe-west1.run.app (herkese açık, `allUsers` → `roles/run.invoker`): yukarıdaki ayarlar + startup CPU boost kapalı. Rolü olmayan servis hesabı `bunker-api-test@siginak-aa1ccd.iam.gserviceaccount.com` ile çalışır (varsayılan Compute hesabı Editor yetkili olduğu için kullanılmaz). Ortam: `APP_ENV=test`, `GCP_PROJECT_ID`, `TRUSTED_PROXY_HOPS=1`, `LOG_LEVEL=info`, `DATABASE_URL` (`backend/.env.cloudrun.yaml` ile; security.md §3).
- Duman testi (2026-11-07): `/health` ve `/health/ready` 200; misafir oturumu 201; `cmd/testbot -create -bots 4 -allow-remote` ile WebSocket üzerinden iki tur oynandı (kart açma, tartışma, oylama, eleme); güvenlik olayı (`auth_failed`) Cloud Logging'de görüldü.
- Servis `bunker-api` (prod) — https://bunker-api-918364240578.europe-west1.run.app (herkese açık): test ile aynı ayarlar; rolsüz servis hesabı `bunker-api@siginak-aa1ccd.iam.gserviceaccount.com`; `APP_ENV=prod`, `DATABASE_URL` yalnızca `bunker_prod_app` rolü (`backend/.env.cloudrun.prod.yaml`, `.env.prod`'dan üretilir). Migration rolü sunucuya verilmez. Duman testi (2026-11-07): `/health`, `/health/ready` 200; misafir oturumu 201, `/v1/me` 200, hesap silme 204. `cmd/testbot` prod'da çalıştırılmaz.
- Bütçe: faturalama hesabında yalnızca bu projeye bakan aylık **200 TL** bütçe; %10/25/50/100 gerçekleşen ve %100 tahmini harcamada e-posta. Uyarıdır, harcamayı durdurmaz.
- Dağıtım (yeni sürüm):
  ```sh
  cd backend
  gcloud builds submit . --project=siginak-aa1ccd --region=europe-west1 \
    --tag=europe-west1-docker.pkg.dev/siginak-aa1ccd/bunker/api:<sürüm>
  gcloud run deploy bunker-api-test --project=siginak-aa1ccd --region=europe-west1 \
    --image=europe-west1-docker.pkg.dev/siginak-aa1ccd/bunker/api:<sürüm>
  ```
  (`run deploy` yalnızca imajı değiştirir; ortam değişkenleri ve diğer ayarlar korunur.)
