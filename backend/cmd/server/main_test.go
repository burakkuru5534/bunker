package main

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/burakkuru5534/bunker/backend/internal/config"
)

// syncBuffer, eşzamanlı yazmalar için kilitli tampon.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// testClient, keep-alive kapalı istemcidir. Varsayılan istemcinin havuzu,
// hiç istek göndermediği bir bağlantıyı açık bırakabilir; http.Server.Shutdown
// böyle "yeni" bağlantıları 5 sn bekler ve kapanış testleri kararsızlaşır.
var testClient = &http.Client{
	Transport: &http.Transport{DisableKeepAlives: true},
	Timeout:   5 * time.Second,
}

func envWith(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// startServer, sunucuyu rastgele bir portta başlatır.
func startServer(t *testing.T, extra map[string]string) (cancel context.CancelFunc, addr string, done <-chan error, out *syncBuffer) {
	t.Helper()
	env := map[string]string{"PORT": "0"} // 0 = işletim sistemi boş port seçer
	for k, v := range extra {
		env[k] = v
	}
	// PORT=0 config doğrulamasından geçmez; testte dinleyiciyi doğrudan
	// seçmek için 1-65535 aralığında boş bir port buluyoruz.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	env["PORT"] = itoa(port)

	ctx, cancelFn := context.WithCancel(context.Background())
	out = &syncBuffer{}
	listening := make(chan net.Addr, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(ctx, envWith(env), out, func(a net.Addr) { listening <- a })
	}()

	select {
	case a := <-listening:
		addr = a.String()
	case err := <-errCh:
		t.Fatalf("sunucu başlamadı: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("sunucu 5 sn içinde dinlemeye başlamadı")
	}
	t.Cleanup(cancelFn)
	return cancelFn, addr, errCh, out
}

func itoa(n int) string {
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func localURL(addr, path string) string {
	_, port, _ := net.SplitHostPort(addr)
	return "http://127.0.0.1:" + port + path
}

func TestServerStartsAndShutsDownGracefully(t *testing.T) {
	cancel, addr, done, out := startServer(t, nil)

	resp, err := testClient.Get(localURL(addr, "/bilinmeyen"))
	if err != nil {
		t.Fatalf("istek başarısız: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("durum = %d, 404 bekleniyordu", resp.StatusCode)
	}

	// Sağlık uç noktaları gerçek sunucuda, sürüm başlığı olmadan çalışır.
	for path, want := range map[string]string{"/health": `{"status":"ok"}`, "/health/ready": `{"status":"ready"}`} {
		resp, err := testClient.Get(localURL(addr, path))
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != want {
			t.Fatalf("%s: %d %s", path, resp.StatusCode, body)
		}
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("temiz kapanış bekleniyordu: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("sunucu 5 sn içinde kapanmadı")
	}

	log := out.String()
	for _, want := range []string{`message="sunucu başladı" env=local`, "httpRequest.status=404", "kapanış başladı", "sunucu durdu"} {
		if !strings.Contains(log, want) {
			t.Errorf("çıktıda %q yok:\n%s", want, log)
		}
	}
}

func TestShutdownWaitsForInFlightRequest(t *testing.T) {
	// Kapanış başladığında işlenmekte olan istek tamamlanmalı.
	entered := make(chan struct{})
	release := make(chan struct{})
	orig := handlerFactory
	handlerFactory = func(config.Config, *slog.Logger, deps) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			close(entered)
			<-release
			w.WriteHeader(http.StatusOK)
		})
	}
	t.Cleanup(func() { handlerFactory = orig })

	cancel, addr, done, _ := startServer(t, map[string]string{"SHUTDOWN_TIMEOUT": "3s"})

	type result struct {
		status int
		err    error
	}
	resCh := make(chan result, 1)
	go func() {
		resp, err := testClient.Get(localURL(addr, "/yavas"))
		if err != nil {
			resCh <- result{err: err}
			return
		}
		_ = resp.Body.Close()
		resCh <- result{status: resp.StatusCode}
	}()

	<-entered
	cancel()

	// İstek bitmeden sunucu kapanmamalı.
	select {
	case err := <-done:
		t.Fatalf("sunucu istek bitmeden kapandı: %v", err)
	case <-time.After(200 * time.Millisecond):
	}

	close(release)
	if r := <-resCh; r.err != nil || r.status != http.StatusOK {
		t.Fatalf("yarım kalan istek tamamlanmadı: %+v", r)
	}
	if err := <-done; err != nil {
		t.Fatalf("temiz kapanış bekleniyordu: %v", err)
	}
}

func TestShutdownTimesOut(t *testing.T) {
	// Kapanış süresi aşılırsa hata döner ve bağlantılar zorla kapatılır.
	entered := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	orig := handlerFactory
	handlerFactory = func(config.Config, *slog.Logger, deps) http.Handler {
		return http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			close(entered)
			<-release
		})
	}
	t.Cleanup(func() { handlerFactory = orig })

	cancel, addr, done, _ := startServer(t, map[string]string{"SHUTDOWN_TIMEOUT": "200ms"})
	go func() {
		if resp, err := testClient.Get(localURL(addr, "/takili")); err == nil {
			_ = resp.Body.Close()
		}
	}()
	<-entered
	cancel()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "kapanış zaman aşımı") {
			t.Fatalf("zaman aşımı hatası bekleniyordu: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("kapanış süresi aşılınca sunucu çıkmadı")
	}
}

func TestRunFailsOnInvalidConfig(t *testing.T) {
	err := run(context.Background(), envWith(map[string]string{"APP_ENV": "staging"}), &syncBuffer{}, nil)
	if err == nil || !strings.Contains(err.Error(), "yapılandırma") {
		t.Fatalf("yapılandırma hatası bekleniyordu: %v", err)
	}
}

func TestRunFailsWhenPortBusy(t *testing.T) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	err = run(context.Background(), envWith(map[string]string{"PORT": itoa(port)}), &syncBuffer{}, nil)
	if err == nil || !strings.Contains(err.Error(), "dinleme") {
		t.Fatalf("port meşgul hatası bekleniyordu: %v", err)
	}
}

func TestLocalEnvReadsDotEnvOnlyLocally(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/.env"
	if err := os.WriteFile(path, []byte("LOG_LEVEL=debug\nPORT=1111\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	get, err := localEnv(envWith(map[string]string{"PORT": "2222"}), path)
	if err != nil || get("LOG_LEVEL") != "debug" || get("PORT") != "2222" {
		t.Fatalf("yerelde .env okunmalı, gerçek ortam öncelikli: %v", err)
	}
	get, err = localEnv(envWith(map[string]string{"APP_ENV": "prod"}), path)
	if err != nil || get("LOG_LEVEL") != "" {
		t.Fatal("bulutta .env okunmamalı")
	}
	if err := os.WriteFile(path, []byte("bozuk satır\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := localEnv(envWith(nil), path); err == nil {
		t.Fatal("bozuk .env hata vermeli")
	}
}

func TestRunFailsWhenDatabaseUnreachable(t *testing.T) {
	var out syncBuffer
	err := run(context.Background(), envWith(map[string]string{
		"PORT":         "8080",
		"DATABASE_URL": "postgres://u:cokgizli@127.0.0.1:1/db?sslmode=disable",
	}), &out, func(net.Addr) { t.Error("veritabanı yokken dinlemeye başlamamalı") })
	if err == nil || !strings.Contains(err.Error(), "veritabanı") {
		t.Fatalf("veritabanı hatası bekleniyordu: %v", err)
	}
	if strings.Contains(err.Error(), "cokgizli") || strings.Contains(out.String(), "cokgizli") {
		t.Fatal("şifre sızmamalı")
	}
}

func TestRunStrictRejectsPlaintextInTest(t *testing.T) {
	err := run(context.Background(), envWith(map[string]string{
		"APP_ENV":      "test",
		"DATABASE_URL": "postgres://u:p@db.example.com/db?sslmode=require",
	}), &syncBuffer{}, nil)
	if err == nil || !strings.Contains(err.Error(), "verify-full") {
		t.Fatalf("test ortamında verify-full zorunlu olmalı: %v", err)
	}
}

// TEST_DATABASE_ADMIN_URL varsa gerçek veritabanıyla açılış + hazırlık.
func TestRunWithDatabaseReady(t *testing.T) {
	admin := os.Getenv("TEST_DATABASE_ADMIN_URL")
	if admin == "" {
		t.Skip("TEST_DATABASE_ADMIN_URL tanımlı değil")
	}
	cancel, addr, done, out := startServer(t, map[string]string{
		"DATABASE_URL": admin,
		"DB_SCHEMA":    "public", // yerel modda yönetici rolüyle; katı denetimler persistence testlerinde
	})
	resp, err := testClient.Get(localURL(addr, "/health/ready"))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("hazır olmalı: %d", resp.StatusCode)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "veritabanı bağlandı") || strings.Contains(out.String(), "postgres:postgres@") {
		t.Fatalf("log: %s", out.String())
	}
}
