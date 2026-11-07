# Sığınak backend imajı (REL-003). Cloud Run'da çalışır (ADR-0011).
#
# Yerelde:  docker build -t bunker-api backend/
# Cloud Run: gcloud run deploy … --source backend/   (Cloud Build bu dosyayı kullanır)
#
# Çok aşamalı: derleme Go imajında, çalışma distroless/static (kabuk ve paket
# yöneticisi yok, root olmayan kullanıcı). İmaj küçük (~20 MB) → hızlı soğuk
# başlatma ve Artifact Registry ücretsiz kotası (0,5 GB) içinde kalır.

FROM golang:1.27.3-bookworm AS build
WORKDIR /src
# Bağımlılıklar ayrı katmanda: yalnızca kod değişince yeniden indirilmez.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/server /app/server
# Supabase kök CA sertifikası (herkese açık; certs/README.md). Çalışma klasörü
# /app olduğundan yereldeki DATABASE_URL aynen çalışır:
#   …?sslmode=verify-full&sslrootcert=certs/supabase-ca.crt
COPY certs/supabase-ca.crt /app/certs/supabase-ca.crt
USER nonroot:nonroot
# Cloud Run PORT'u verir (varsayılan 8080).
EXPOSE 8080
ENTRYPOINT ["/app/server"]
