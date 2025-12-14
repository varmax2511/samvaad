package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type Handler struct {
	store *UserStore
}

func NewHandler(store *UserStore) *Handler {
	return &Handler{store: store}
}

type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token  string `json:"token,omitempty"`
	UserId string `json:"userId,omitempty"`
	Error  string `json:"error,omitempty"`
}

func writeJson(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJson(w, http.StatusBadRequest, AuthResponse{Error: "invalid request"})
		slog.Error("invalid request when decoding request body", "body", r.Body)
		return
	}

	user, err := h.store.Create(req.Username, req.Password)
	if err != nil {
		writeJson(w, http.StatusConflict, AuthResponse{Error: err.Error()})
		slog.Error("conflict: failed to create user enty", "username", req.Username, "error", err.Error())
		return
	}

	token, err := GenerateToken(user.Id, user.Username)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, AuthResponse{Error: "internal error: failed to generate token"})
		slog.Error("internal error: failed to generate token", "username", req.Username)
		return
	}
	writeJson(w, http.StatusCreated, AuthResponse{Token: token, UserId: user.Id})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req AuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJson(w, http.StatusBadRequest, AuthResponse{Error: "invalid request"})
		slog.Error("login failed: invalid request", "body", r.Body)
		return
	}

	user, err := h.store.GetByUsername(req.Username)
	// whether username doesn't exist or its bad password, in both cases return same code and message
	if err != nil || !CheckPassword(req.Password, user.PasswordHash) {
		writeJson(w, http.StatusUnauthorized, AuthResponse{Error: "invalid credentials"})
		slog.Error("login failed: invalid credentials", "body", r.Body)
		return
	}

	token, err := GenerateToken(user.Id, user.Username)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, AuthResponse{Error: "internal error: failed to generate token"})
		slog.Error("internal error: failed to generate token", "username", req.Username)
		return
	}
	writeJson(w, http.StatusOK, AuthResponse{Token: token, UserId: user.Id})
}
