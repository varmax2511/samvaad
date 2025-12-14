package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/varmax2511/samvaad/backend/internal/auth"
	"github.com/varmax2511/samvaad/backend/internal/signaling"
)

func TestUpgraderCheckOriginAllows(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	if ok := upgrader.CheckOrigin(req); !ok {
		t.Fatalf("expected upgrader.CheckOrigin to allow request, got false")
	}
}

func TestHandleWebSocket_UpgradesConnection_WithValidToken(t *testing.T) {
	// create and run hub
	hub := signaling.NewHub()
	go hub.Run()

	// generate a valid token
	token, err := auth.GenerateToken("test-user-id", "testuser")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	// create a mux that uses the handleWebSocket from main.go
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// build ws URL with token
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=" + token

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

func TestHandleWebSocket_RejectsWithoutToken(t *testing.T) {
	hub := signaling.NewHub()
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// build ws URL without token
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	time.Sleep(10 * time.Millisecond)

	dialer := websocket.DefaultDialer
	_, resp, err := dialer.Dial(wsURL, nil)

	// Should fail to connect
	if err == nil {
		t.Fatal("expected error when connecting without token")
	}

	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
	}
}

func TestHandleWebSocket_RejectsInvalidToken(t *testing.T) {
	hub := signaling.NewHub()
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// build ws URL with invalid token
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=invalid-token"

	time.Sleep(10 * time.Millisecond)

	dialer := websocket.DefaultDialer
	_, resp, err := dialer.Dial(wsURL, nil)

	// Should fail to connect
	if err == nil {
		t.Fatal("expected error when connecting with invalid token")
	}

	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, resp.StatusCode)
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

// Helper to create a test server with all endpoints
func createTestServer() *httptest.Server {
	hub := signaling.NewHub()
	go hub.Run()

	userStore := auth.NewUserStore()
	authHandler := auth.NewHandler(userStore)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/register", authHandler.Register)
	mux.HandleFunc("/auth/login", authHandler.Login)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	return httptest.NewServer(mux)
}

func TestAuthEndpoints_Register(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	body := `{"username": "testuser", "password": "password123"}`
	resp, err := http.Post(server.URL+"/auth/register", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("POST /auth/register failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var authResp auth.AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if authResp.Token == "" {
		t.Error("expected non-empty token")
	}

	if authResp.UserId == "" {
		t.Error("expected non-empty userId")
	}
}

func TestAuthEndpoints_Login(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	// First register
	regBody := `{"username": "testuser", "password": "password123"}`
	regResp, err := http.Post(server.URL+"/auth/register", "application/json", bytes.NewBufferString(regBody))
	if err != nil {
		t.Fatalf("POST /auth/register failed: %v", err)
	}
	regResp.Body.Close()

	// Then login
	loginBody := `{"username": "testuser", "password": "password123"}`
	loginResp, err := http.Post(server.URL+"/auth/login", "application/json", bytes.NewBufferString(loginBody))
	if err != nil {
		t.Fatalf("POST /auth/login failed: %v", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, loginResp.StatusCode)
	}

	var authResp auth.AuthResponse
	if err := json.NewDecoder(loginResp.Body).Decode(&authResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if authResp.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestAuthEndpoints_LoginWrongPassword(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	// First register
	regBody := `{"username": "testuser", "password": "password123"}`
	regResp, err := http.Post(server.URL+"/auth/register", "application/json", bytes.NewBufferString(regBody))
	if err != nil {
		t.Fatalf("POST /auth/register failed: %v", err)
	}
	regResp.Body.Close()

	// Login with wrong password
	loginBody := `{"username": "testuser", "password": "wrongpassword"}`
	loginResp, err := http.Post(server.URL+"/auth/login", "application/json", bytes.NewBufferString(loginBody))
	if err != nil {
		t.Fatalf("POST /auth/login failed: %v", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, loginResp.StatusCode)
	}
}

func TestFullAuthFlow_RegisterLoginConnect(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	// Step 1: Register
	regBody := `{"username": "testuser", "password": "password123"}`
	regResp, err := http.Post(server.URL+"/auth/register", "application/json", bytes.NewBufferString(regBody))
	if err != nil {
		t.Fatalf("POST /auth/register failed: %v", err)
	}

	var regAuthResp auth.AuthResponse
	json.NewDecoder(regResp.Body).Decode(&regAuthResp)
	regResp.Body.Close()

	if regAuthResp.Token == "" {
		t.Fatal("registration did not return token")
	}

	// Step 2: Login
	loginBody := `{"username": "testuser", "password": "password123"}`
	loginResp, err := http.Post(server.URL+"/auth/login", "application/json", bytes.NewBufferString(loginBody))
	if err != nil {
		t.Fatalf("POST /auth/login failed: %v", err)
	}

	var loginAuthResp auth.AuthResponse
	json.NewDecoder(loginResp.Body).Decode(&loginAuthResp)
	loginResp.Body.Close()

	if loginAuthResp.Token == "" {
		t.Fatal("login did not return token")
	}

	// Step 3: Connect to WebSocket with token
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=" + loginAuthResp.Token

	time.Sleep(10 * time.Millisecond)

	dialer := websocket.DefaultDialer
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		body := ""
		if resp != nil && resp.Body != nil {
			b, _ := io.ReadAll(resp.Body)
			body = string(b)
		}
		t.Fatalf("failed to dial websocket: %v, body: %s", err, body)
	}

	if conn == nil {
		t.Fatal("expected non-nil websocket connection")
	}

	conn.Close()
}

func TestFullAuthFlow_RegisterAndConnectDirectly(t *testing.T) {
	server := createTestServer()
	defer server.Close()

	// Register and use token directly (skip login)
	regBody := `{"username": "testuser", "password": "password123"}`
	regResp, err := http.Post(server.URL+"/auth/register", "application/json", bytes.NewBufferString(regBody))
	if err != nil {
		t.Fatalf("POST /auth/register failed: %v", err)
	}

	var authResp auth.AuthResponse
	json.NewDecoder(regResp.Body).Decode(&authResp)
	regResp.Body.Close()

	// Connect to WebSocket with registration token
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=" + authResp.Token

	time.Sleep(10 * time.Millisecond)

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect with registration token: %v", err)
	}

	conn.Close()
}
