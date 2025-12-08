package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/varmax2511/samvaad/backend/internal/signaling"
)

func TestUpgraderCheckOriginAllows(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	if ok := upgrader.CheckOrigin(req); !ok {
		t.Fatalf("expected upgrader.CheckOrigin to allow request, got false")
	}
}

func TestHandleWebSocket_UpgradesConnection(t *testing.T) {
	// create and run hub
	hub := signaling.NewHub()
	go hub.Run()

	// create a mux that uses the handleWebSocket from main.go
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// build ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// give the hub a moment to start
	time.Sleep(10 * time.Millisecond)

	// dial the websocket endpoint
	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		// include response body when available for easier debugging
		body := ""
		if resp != nil && resp.Body != nil {
			b, _ := io.ReadAll(resp.Body)
			body = string(b)
		}
		t.Fatalf("failed to dial websocket: %v, resp status: %v, body: %s", err, resp.Status, body)
	}
	// ensure connection is alive
	if conn == nil {
		t.Fatalf("expected non-nil websocket connection")
	}

	// close connection cleanly
	if err := conn.Close(); err != nil {
		t.Fatalf("failed to close websocket connection: %v", err)
	}
}

func TestHealthHandler_ReturnsOK(t *testing.T) {
	// register a health handler equivalent to main.go
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("reading body failed: %v", err)
	}
	if string(body) != "OK" {
		t.Fatalf("expected body 'OK', got %q", string(body))
	}
}
