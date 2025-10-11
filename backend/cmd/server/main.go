package main

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/varmax2511/samvaad/backend/internal/logger"
	"github.com/varmax2511/samvaad/backend/internal/signaling"
)

// upgrades the HTTP server connection to the WebSocket protocol
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: Implement proper CORS
		return true
	},
}

func main() {
	appLogger := logger.NewLogger()
	slog.SetDefault(appLogger)

	slog.Info("starting Samvaad server ....")

	hub := signaling.NewHub()
	go hub.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	slog.Info("server starting on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		slog.Error("error while starting the HTTP server", "error", err.Error())
	}
}

func handleWebSocket(hub *signaling.Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("error during connection upgrade to websocket", "error", err.Error())
	}

	clientId := uuid.New().String()
	client := signaling.NewClient(clientId, hub, conn)

	hub.Register <- client
	slog.Debug("registered new client", "clientId", clientId)

	go client.WriteMessages()
	go client.ReadMessages()
}
