package signaling

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 //512KB
)

type Client struct {
	ID       string
	UserID   string
	Username string
	RoomID   string
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan []byte
}

func NewClient(id string, userId string, username string, hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		ID:       id,
		UserID:   userId,
		Username: username,
		Hub:      hub,
		Conn:     conn,
		Send:     make(chan []byte, 256),
	}
}

func (c *Client) ReadMessages() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	// set read timeout
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(appData string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// read messages in an infinite loop
	for {
		_, messageData, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("read message connection error: %v", "error", err)
			}
			break
		}

		var message Message
		if err := json.Unmarshal(messageData, &message); err != nil {
			slog.Error("error unmarshaling messag: %v", "error", err)
			continue
		}

		message.From = c.ID
		c.Hub.Broadcast <- &message
	}
}

func (c *Client) WriteMessages() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send each message as a separate WebSocket frame
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				slog.Error("error while writing message", slog.Any("err", err))
				return
			}

			// Send any queued messages as separate frames
			n := len(c.Send)
			for i := 0; i < n; i++ {
				msg := <-c.Send
				if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					slog.Error("error while writing queued message", slog.Any("err", err))
					return
				}
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Error("error while writing ping message", slog.Any("err", err))
				return
			}
		}
	}

}
