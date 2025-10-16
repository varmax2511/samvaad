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
	ID     string
	RoomID string
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
}

func NewClient(id string, hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		ID:   id,
		Hub:  hub,
		Conn: conn,
		Send: make(chan []byte, 256),
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
				slog.Error("read message connection error: %v", err)
			}
			break
		}

		var message Message
		if err := json.Unmarshal(messageData, &message); err != nil {
			slog.Error("error unmarshaling messag: %v", err)
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

			writer, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				slog.Error("errow while instantiating a writer for messages %v", err)
			}
			writer.Write(message)

			// add queued messages
			for i := 0; i < len(c.Send); i++ {
				writer.Write([]byte{'\n'})
				writer.Write(<-c.Send)
			}

			if err := writer.Close(); err != nil {
				slog.Error("error while closing the writer", slog.Any("err", err))
				return
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
