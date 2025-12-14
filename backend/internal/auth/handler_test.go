package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler_Register_Success(t *testing.T) {
	store := NewUserStore()
	handler := NewHandler(store)

	body := `{"username": "testuser", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var resp AuthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Token == "" {
		t.Error("expected non-empty token")
	}

	if resp.UserId == "" {
		t.Error("expected non-empty userId")
	}

	if resp.Error != "" {
		t.Errorf("expected no error, got %q", resp.Error)
	}
}

func TestHandler_Register_DuplicateUsername(t *testing.T) {
	store := NewUserStore()
	handler := NewHandler(store)

	// First registration
	body := `{"username": "testuser", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("first registration failed with status %d", w.Code)
	}

	// Second registration with same username
	req = httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.Register(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, w.Code)
	}

	var resp AuthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == "" {
		t.Error("expected error message for duplicate username")
	}
}

func TestHandler_Register_InvalidJSON(t *testing.T) {
	store := NewUserStore()
	handler := NewHandler(store)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandler_Register_EmptyBody(t *testing.T) {
	store := NewUserStore()
	handler := NewHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandler_Login_Success(t *testing.T) {
	store := NewUserStore()
	handler := NewHandler(store)

	// First register a user
	_, err := store.Create("testuser", "password123")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Now login
	body := `{"username": "testuser", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp AuthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Token == "" {
		t.Error("expected non-empty token")
	}

	if resp.UserId == "" {
		t.Error("expected non-empty userId")
	}

	if resp.Error != "" {
		t.Errorf("expected no error, got %q", resp.Error)
	}
}

func TestHandler_Login_WrongPassword(t *testing.T) {
	store := NewUserStore()
	handler := NewHandler(store)

	// First register a user
	_, err := store.Create("testuser", "password123")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Login with wrong password
	body := `{"username": "testuser", "password": "wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var resp AuthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error == "" {
		t.Error("expected error message for wrong password")
	}
}

func TestHandler_Login_NonExistentUser(t *testing.T) {
	store := NewUserStore()
	handler := NewHandler(store)

	body := `{"username": "nonexistent", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestHandler_Login_InvalidJSON(t *testing.T) {
	store := NewUserStore()
	handler := NewHandler(store)

	body := `{invalid json}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandler_Login_EmptyBody(t *testing.T) {
	store := NewUserStore()
	handler := NewHandler(store)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestHandler_Register_ThenLogin(t *testing.T) {
	store := NewUserStore()
	handler := NewHandler(store)

	// Register
	regBody := `{"username": "testuser", "password": "password123"}`
	regReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(regBody))
	regReq.Header.Set("Content-Type", "application/json")
	regW := httptest.NewRecorder()
	handler.Register(regW, regReq)

	if regW.Code != http.StatusCreated {
		t.Fatalf("registration failed with status %d", regW.Code)
	}

	var regResp AuthResponse
	json.NewDecoder(regW.Body).Decode(&regResp)

	// Login
	loginBody := `{"username": "testuser", "password": "password123"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	handler.Login(loginW, loginReq)

	if loginW.Code != http.StatusOK {
		t.Fatalf("login failed with status %d", loginW.Code)
	}

	var loginResp AuthResponse
	json.NewDecoder(loginW.Body).Decode(&loginResp)

	// Both should return same userId
	if regResp.UserId != loginResp.UserId {
		t.Errorf("expected same userId, got register=%q login=%q", regResp.UserId, loginResp.UserId)
	}

	// Both tokens should be valid (we don't check if they're different
	// since tokens generated in the same second will be identical)
	regClaims, err := ValidateToken(regResp.Token)
	if err != nil {
		t.Errorf("register token validation failed: %v", err)
	}

	loginClaims, err := ValidateToken(loginResp.Token)
	if err != nil {
		t.Errorf("login token validation failed: %v", err)
	}

	if regClaims.UserId != loginClaims.UserId {
		t.Error("token claims should have same userId")
	}
}
