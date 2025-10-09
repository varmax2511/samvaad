package signaling

import (
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

func (c *Client) ReadMessages() {}

func (c *Client) WriteMessages() {}
