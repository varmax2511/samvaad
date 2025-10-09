package signaling

import (
	"encoding/json"
	"log/slog"
	"sync"
)

type Hub struct {
	// registered clients
	Clients map[string]*Client

	// roomID -> map of client IDs
	Rooms map[string]map[string]*Client

	// Channels
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *Message

	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[string]*Client),
		Rooms:      make(map[string]map[string]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *Message),
	}
}

func (h *Hub) Run() {
	// todo
}

func (h *Hub) handleMessage(message *Message) {

}

func (h *Hub) handleJoin(message *Message) {

}

func (h *Hub) handleLeave(message *Message) {

}

func (h *Hub) leaveRoom(client *Client) {

}

func (h *Hub) handleChat(message *Message) {}

func (h *Hub) sendToClient(client *Client, message *Message) {}

func (h *Hub) sendError(client *Client, err error) {
	message := Message{
		Type: MessageTypeError,
	}
	payload := map[string]string{"message": err.Error()}
	payloadBytes, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		slog.Error(marshalErr.Error())
	}

	message.Payload = payloadBytes
	h.sendToClient(client, &message)
}
