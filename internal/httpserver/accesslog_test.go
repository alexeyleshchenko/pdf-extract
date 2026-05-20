package httpserver

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccessLog_skipsHealthAtInfo(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := AccessLog(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/health", "/v1/health"} {
		buf.Reset()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("path %s: status %d", path, rec.Code)
		}
		if strings.TrimSpace(buf.String()) != "" {
			t.Fatalf("path %s: expected no Info log, got %q", path, buf.String())
		}
	}
}

func TestAccessLog_logsNonHealth(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := AccessLog(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/v1/process", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if buf.Len() == 0 {
		t.Fatal("expected http_request log line")
	}
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log JSON: %v", err)
	}
	if entry["msg"] != "http_request" {
		t.Fatalf("msg = %v, want http_request", entry["msg"])
	}
	if entry["path"] != "/v1/process" {
		t.Fatalf("path = %v", entry["path"])
	}
}

func TestAccessLog_healthAtDebug(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	handler := AccessLog(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !strings.Contains(buf.String(), "http_request") {
		t.Fatalf("expected debug http_request log, got %q", buf.String())
	}
}
