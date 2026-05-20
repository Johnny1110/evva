package hooks

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunHTTP_HappyPath(t *testing.T) {
	var got atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"x":1}` {
			t.Errorf("body=%s", body)
		}
		if r.Header.Get("X-Custom") != "yes" {
			t.Errorf("X-Custom missing")
		}
		got.Add(1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cmd := Command{
		Type:    TypeHTTP,
		URL:     srv.URL,
		Headers: map[string]string{"X-Custom": "yes"},
	}
	if err := runHTTP(context.Background(), quietLogger(), cmd, []byte(`{"x":1}`), 2*time.Second); err != nil {
		t.Fatalf("err=%v", err)
	}
	if got.Load() != 1 {
		t.Errorf("server hit count=%d", got.Load())
	}
}

func TestRunHTTP_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	cmd := Command{Type: TypeHTTP, URL: srv.URL}
	if err := runHTTP(context.Background(), quietLogger(), cmd, nil, 2*time.Second); err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestRunHTTP_EmptyURL(t *testing.T) {
	if err := runHTTP(context.Background(), quietLogger(), Command{Type: TypeHTTP}, nil, time.Second); err == nil {
		t.Fatal("expected error for empty URL")
	}
}

func TestRunHTTP_CustomMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("method=%s", r.Method)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cmd := Command{Type: TypeHTTP, URL: srv.URL, Method: "PUT"}
	if err := runHTTP(context.Background(), quietLogger(), cmd, nil, 2*time.Second); err != nil {
		t.Fatalf("err=%v", err)
	}
}
