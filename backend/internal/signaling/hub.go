package signaling

import (
	"encoding/json"
	"errors"
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
	// support register, unregister and message broadcast in an infinite loop
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client.ID] = client
			h.mu.Unlock()
			slog.Info("client registered", "clientID", client.ID)

		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client.ID]; ok {
				// leave room if in one
				if client.RoomID != "" {
					h.leaveRoom(client)
				}
				delete(h.Clients, client.ID)
				close(client.Send)
				slog.Info("client unregistered", "clientID", client.ID)
			}

		case message := <-h.Broadcast:
			h.handleMessage(message)
		}
	}
}

func (h *Hub) handleMessage(message *Message) {
	switch message.Type {
	case MessageTypeJoin:
		h.handleJoin(message)
	case MessageTypeLeave:
		h.handleLeave(message)
	case MessageTypeAnswer, MessageTypeIceCandidate, MessageTypeOffer:
		h.handleSignaling(message)
	case MessageTypeChat:
		h.handleChat(message)
	default:
		slog.Error("unknown message type", "type", string(message.Type))
		panic("unknown message type: " + message.Type)
	}
}

func (h *Hub) handleJoin(message *Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, ok := h.Clients[message.From]
	if !ok {
		slog.Error("join request initiated by a non-existent client...returning")
		return
	}

	// check if room exists, can be a new room
	roomId := message.RoomID
	if _, exists := h.Rooms[roomId]; !exists {
		h.Rooms[roomId] = make(map[string]*Client)
	}

	// check if room is full, for phase1 we are restricting to two users
	if len(h.Rooms[roomId]) >= 2 {
		slog.Debug("room is full for roomId", "roomId", roomId)
		h.sendError(client, errors.New("room is full"))
		return
	}

	// add the client now to the room
	h.Rooms[roomId][client.ID] = client
	client.RoomID = roomId

	// list other users in the room
	otherUsers := []string{}
	for id := range h.Rooms[roomId] {
		if id != client.ID {
			otherUsers = append(otherUsers, id)
		}
	}

	// send confirmation to the requesting client
	joinedMessage := Message{
		Type:   MessageTypeUserJoined,
		RoomID: roomId,
		From:   client.ID,
	}
	payload := map[string]interface{}{
		"userId":     client.ID,
		"otherUsers": otherUsers,
	}
	payloadBytes, _ := json.Marshal(payload)
	joinedMessage.Payload = payloadBytes
	slog.Debug("sending join confirmation to requesting client")
	h.sendToClient(client, &joinedMessage)

	// notify other users in the room
	userJoinedMessage := Message{
		Type:   MessageTypeUserJoined,
		RoomID: roomId,
		From:   client.ID,
	}

	userJoinedPayload := map[string]interface{}{
		"userId": client.ID,
	}
	userJoinedPayloadBytes, _ := json.Marshal(userJoinedPayload)
	userJoinedMessage.Payload = userJoinedPayloadBytes
	h.broadcastToRoom(roomId, &userJoinedMessage, map[string]bool{client.ID: true})
	slog.Info("client joined room", "userJoinedMessage", userJoinedMessage, "clientID", client.ID)
}

// function send message to all clients in the room except the ones listed under excludeClientId
func (h *Hub) broadcastToRoom(roomId string, message *Message, excludeClientId map[string]bool) {
	if _, exists := h.Rooms[roomId]; !exists {
		slog.Error("broadcast request sent for a room that doesn't exist", "roomId", roomId)
		return
	}

	room := h.Rooms[roomId]
	messageBytes, _ := json.Marshal(message)
	for id, client := range room {
		if _, exists := excludeClientId[id]; !exists {
			select {
			case client.Send <- messageBytes:
			default:
				close(client.Send)
				delete(h.Clients, client.ID)
				delete(room, id)
			}
		}
	}
}

func (h *Hub) handleLeave(message *Message) {
	h.mu.Lock()
	defer h.mu.Unlock()

	client, ok := h.Clients[message.From]
	if !ok {
		slog.Warn("leave room request sent by a non existent client", "clientID", message.From)
		return
	}

	h.leaveRoom(client)
}

func (h *Hub) leaveRoom(client *Client) {
	if client.RoomID == "" {
		client.Hub.sendError(client, errors.New("you are not in a room to leave"))
		return
	}

	roomId := client.RoomID

	// remove client from room
	if room, exists := h.Rooms[roomId]; exists {
		delete(room, roomId)

		// notify other users
		leftMessage := Message{
			Type:   MessageTypeLeave,
			RoomID: roomId,
			From:   client.ID,
		}
		leftMessagePayload := map[string]interface{}{
			"userId": client.ID,
		}
		leftMessagePayloadBytes, _ := json.Marshal(leftMessagePayload)
		leftMessage.Payload = leftMessagePayloadBytes
		h.broadcastToRoom(roomId, &leftMessage, map[string]bool{client.ID: true})

		// delete the room if its empty
		if len(room) == 0 {
			delete(h.Rooms, roomId)
		}
	}
	client.RoomID = ""
	slog.Info("client % left room %", client.ID, roomId)
}

func (h *Hub) handleSignaling(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// forward signaling message to peer
	if message.To != "" {
		if targetClient, ok := h.Clients[message.To]; ok {
			h.sendToClient(targetClient, message)
		}
	}
}

func (h *Hub) handleChat(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// broadcast message to room
	if message.RoomID != "" {
		h.broadcastToRoom(message.RoomID, message, nil)
	}
}

func (h *Hub) sendToClient(client *Client, message *Message) {
	messageBytes, _ := json.Marshal(message)
	select {
	case client.Send <- messageBytes:
	default:
		close(client.Send)
		delete(h.Clients, client.ID)
	}
}

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
