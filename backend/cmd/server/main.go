package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/varmax2511/samvaad/backend/internal/auth"
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
	// Command line flags
	port := flag.String("port", "8080", "Server port")
	certFile := flag.String("cert", "", "Path to TLS certificate file")
	keyFile := flag.String("key", "", "Path to TLS key file")
	flag.Parse()

	appLogger := logger.NewLogger()
	slog.SetDefault(appLogger)

	slog.Info("starting Samvaad server ....")

	hub := signaling.NewHub()
	go hub.Run()

	// Setup auth
	userStore := auth.NewUserStore()
	authHandler := auth.NewHandler(userStore)

	// Auth endpoints
	http.HandleFunc("/auth/register", authHandler.Register)
	http.HandleFunc("/auth/login", authHandler.Login)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "../../room.html")
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := ":" + *port

	// Check if certs exist in default location
	if *certFile == "" && *keyFile == "" {
		defaultCert := "../../certs/cert.pem"
		defaultKey := "../../certs/key.pem"
		if _, err := os.Stat(defaultCert); err == nil {
			if _, err := os.Stat(defaultKey); err == nil {
				*certFile = defaultCert
				*keyFile = defaultKey
			}
		}
	}

	// Start HTTPS if certs are available, otherwise HTTP
	if *certFile != "" && *keyFile != "" {
		slog.Info("server starting with HTTPS", "port", *port)
		slog.Info("open https://localhost:" + *port + " in your browser")
		if err := http.ListenAndServeTLS(addr, *certFile, *keyFile, nil); err != nil {
			slog.Error("error while starting HTTPS server", "error", err.Error())
		}
	} else {
		slog.Info("server starting with HTTP (no TLS certs found)", "port", *port)
		slog.Info("open http://localhost:" + *port + " in your browser")
		slog.Info("TIP: For camera access on other devices, generate certs with: ./scripts/generate-certs.sh")
		if err := http.ListenAndServe(addr, nil); err != nil {
			slog.Error("error while starting HTTP server", "error", err.Error())
		}
	}
}

func handleWebSocket(hub *signaling.Hub, w http.ResponseWriter, r *http.Request) {
	// extract token from query param
	tokenString := r.URL.Query().Get("token")
	if tokenString == "" {
		http.Error(w, "missing auth token", http.StatusUnauthorized)
		return
	}

	// validate token
	claims, err := auth.ValidateToken(tokenString)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("error during connection upgrade to websocket", "error", err.Error())
		return
	}

	// client with authenticated user info
	clientId := uuid.New().String()
	client := signaling.NewClient(clientId, claims.UserId, claims.Username, hub, conn)

	hub.Register <- client
	slog.Debug("registered new client", "clientId", clientId)

	go client.WriteMessages()
	go client.ReadMessages()
}
