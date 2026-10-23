package api

import (
	"context"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/ivantit66/onebase/internal/auth"
	"github.com/ivantit66/onebase/internal/dsl/interpreter"
	"github.com/ivantit66/onebase/internal/runtime"
	"github.com/ivantit66/onebase/internal/storage"
	"github.com/ivantit66/onebase/internal/ui"
)

// serveH2CTestServer поднимает реальный listener и обслуживает сервер до конца
// теста, возвращая host:port. ONEBASE_H2C нужно выставить ДО вызова: New читает
// его при конструировании (см. h2cEnabled).
func serveH2CTestServer(t *testing.T) string {
	t.Helper()
	ctx := context.Background()
	db, err := storage.ConnectSQLite(ctx, filepath.Join(t.TempDir(), "h2c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	authRepo := auth.NewRepo(db)
	if err := authRepo.EnsureSchema(ctx); err != nil {
		t.Fatal(err)
	}
	srv := newTestServer(runtime.NewRegistry(), db, interpreter.New(), authRepo, "127.0.0.1", 0, ui.Config{}, nil)
	t.Cleanup(srv.frontend.(*ui.Server).Close)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.srv.Serve(ln) }()
	t.Cleanup(func() {
		sctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(sctx)
	})
	return ln.Addr().String()
}

// h2cClient — HTTP/2-клиент по cleartext (prior knowledge): включён только
// UnencryptedHTTP2, без HTTP/1.1 fallback, который мог бы скрыть отказ h2c.
func h2cClient(t *testing.T) *http.Client {
	t.Helper()
	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	transport := &http.Transport{Protocols: protocols}
	t.Cleanup(transport.CloseIdleConnections)
	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
	}
}

// При ONEBASE_H2C=1 listener обслуживает HTTP/2 без TLS: h2c-клиент получает
// ответ по HTTP/2 (план 111, P2-1).
func TestH2CUpstreamEnabled(t *testing.T) {
	t.Setenv("ONEBASE_H2C", "1")
	addr := serveH2CTestServer(t)

	resp, err := h2cClient(t).Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("h2c GET /health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.ProtoMajor != 2 {
		t.Fatalf("proto = %q, ожидали HTTP/2", resp.Proto)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, ожидали 200", resp.StatusCode)
	}
}

// Без ONEBASE_H2C сервер остаётся HTTP/1.1-only: тот же h2c-клиент не должен
// получить HTTP/2; обычный HTTP/1.1-запрос остаётся успешным.
func TestH2CUpstreamDisabledByDefault(t *testing.T) {
	t.Setenv("ONEBASE_H2C", "") // детерминизм при унаследованном окружении CI
	addr := serveH2CTestServer(t)

	resp, err := h2cClient(t).Get("http://" + addr + "/health")
	if err == nil {
		defer func() { _ = resp.Body.Close() }()
		if resp.ProtoMajor == 2 {
			t.Fatal("сервер отдал HTTP/2, хотя h2c выключен")
		}
	}

	transport := &http.Transport{}
	t.Cleanup(transport.CloseIdleConnections)
	client := &http.Client{Timeout: 5 * time.Second, Transport: transport}
	resp, err = client.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("HTTP/1.1 GET /health: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.ProtoMajor != 1 || resp.StatusCode != http.StatusOK {
		t.Fatalf("proto = %q, status = %d, ожидали HTTP/1.1 и 200", resp.Proto, resp.StatusCode)
	}
}
