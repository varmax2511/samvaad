package signaling

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func waitFor(t *testing.T, timeout time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.After(timeout)
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("condition not met within %v", timeout)
		case <-tick.C:
			if fn() {
				return
			}
		}
	}
}

func readMessage(t *testing.T, ch chan []byte, timeout time.Duration) Message {
	t.Helper()
	select {
	case b, ok := <-ch:
		if !ok {
			t.Fatalf("channel closed while waiting for message")
		}
		var m Message
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("failed to unmarshal message: %v", err)
		}
		return m
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for message")
	}
	return Message{}
}

func TestRegisterAndUnregisterClient(t *testing.T) {
	h := NewHub()
	go h.Run()

	c := &Client{
		ID:   "client1",
		Send: make(chan []byte, 2),
		Hub:  h,
	}

	// register
	h.Register <- c

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok := h.Clients[c.ID]
		h.mu.RUnlock()
		return ok
	})

	// unregister
	h.Unregister <- c

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok := h.Clients[c.ID]
		h.mu.RUnlock()
		return !ok
	})
}

func TestJoinRoomAndNotifyOtherUser(t *testing.T) {
	h := NewHub()
	go h.Run()

	c1 := &Client{ID: "u1", Send: make(chan []byte, 4), Hub: h}
	c2 := &Client{ID: "u2", Send: make(chan []byte, 4), Hub: h}

	h.Register <- c1
	h.Register <- c2

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok1 := h.Clients[c1.ID]
		_, ok2 := h.Clients[c2.ID]
		h.mu.RUnlock()
		return ok1 && ok2
	})

	// u1 joins room "roomA"
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: c1.ID, RoomID: "roomA"}

	// u1 should receive join confirmation
	m1 := readMessage(t, c1.Send, 500*time.Millisecond)
	if m1.Type != MessageTypeUserJoined {
		t.Fatalf("expected type %s got %s", MessageTypeUserJoined, m1.Type)
	}
	// payload should contain userId == u1 and otherUsers as empty array
	var payload1 map[string]interface{}
	if err := json.Unmarshal(m1.Payload, &payload1); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload1["userId"] != c1.ID {
		t.Fatalf("expected userId %s got %v", c1.ID, payload1["userId"])
	}

	// u2 joins same room
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: c2.ID, RoomID: "roomA"}

	// u2 should get confirmation
	m2 := readMessage(t, c2.Send, 500*time.Millisecond)
	if m2.Type != MessageTypeUserJoined {
		t.Fatalf("expected type %s got %s", MessageTypeUserJoined, m2.Type)
	}
	// u1 should get notified that u2 joined
	notify := readMessage(t, c1.Send, 500*time.Millisecond)
	if notify.Type != MessageTypeUserJoined {
		t.Fatalf("expected notify type %s got %s", MessageTypeUserJoined, notify.Type)
	}
	var notifyPayload map[string]interface{}
	if err := json.Unmarshal(notify.Payload, &notifyPayload); err != nil {
		t.Fatalf("failed to unmarshal notify payload: %v", err)
	}
	if notifyPayload["userId"] != c2.ID {
		t.Fatalf("expected notify userId %s got %v", c2.ID, notifyPayload["userId"])
	}

	// verify room membership
	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		room, ok := h.Rooms["roomA"]
		h.mu.RUnlock()
		if !ok {
			return false
		}
		_, ok1 := room[c1.ID]
		_, ok2 := room[c2.ID]
		return ok1 && ok2
	})
}

func TestRoomFullRejectsThirdJoin(t *testing.T) {
	h := NewHub()
	go h.Run()

	a := &Client{ID: "a", Send: make(chan []byte, 4), Hub: h}
	b := &Client{ID: "b", Send: make(chan []byte, 4), Hub: h}
	c := &Client{ID: "c", Send: make(chan []byte, 4), Hub: h}

	h.Register <- a
	h.Register <- b
	h.Register <- c

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, okA := h.Clients[a.ID]
		_, okB := h.Clients[b.ID]
		_, okC := h.Clients[c.ID]
		h.mu.RUnlock()
		return okA && okB && okC
	})

	// a and b join
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: a.ID, RoomID: "roomFull"}
	_ = readMessage(t, a.Send, 500*time.Millisecond)
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: b.ID, RoomID: "roomFull"}
	_ = readMessage(t, b.Send, 500*time.Millisecond)
	// c tries to join and should receive an error
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: c.ID, RoomID: "roomFull"}
	errMsg := readMessage(t, c.Send, 500*time.Millisecond)
	if errMsg.Type != MessageTypeError {
		t.Fatalf("expected error message type, got %s", errMsg.Type)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(errMsg.Payload, &errPayload); err != nil {
		t.Fatalf("failed to unmarshal error payload: %v", err)
	}
	if !strings.Contains(errPayload["message"], "room is full") {
		t.Fatalf("expected room is full error, got %v", errPayload["message"])
	}
}

func TestSignalingForwarding(t *testing.T) {
	h := NewHub()
	go h.Run()

	sender := &Client{ID: "send", Send: make(chan []byte, 4), Hub: h}
	recv := &Client{ID: "recv", Send: make(chan []byte, 4), Hub: h}

	h.Register <- sender
	h.Register <- recv

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok1 := h.Clients[sender.ID]
		_, ok2 := h.Clients[recv.ID]
		h.mu.RUnlock()
		return ok1 && ok2
	})

	// send an offer from sender to recv
	msg := &Message{
		Type:   MessageTypeOffer,
		From:   sender.ID,
		To:     recv.ID,
		RoomID: "",
		Payload: func() []byte {
			b, _ := json.Marshal(map[string]string{"sdp": "dummy"})
			return b
		}(),
	}
	h.Broadcast <- msg

	// recv should get the message forwarded
	got := readMessage(t, recv.Send, 500*time.Millisecond)
	if got.Type != MessageTypeOffer {
		t.Fatalf("expected offer type, got %s", got.Type)
	}
	if got.From != sender.ID {
		t.Fatalf("expected from %s got %s", sender.ID, got.From)
	}
	if got.To != recv.ID {
		t.Fatalf("expected to %s got %s", recv.ID, got.To)
	}
	var payload map[string]string
	if err := json.Unmarshal(got.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload["sdp"] != "dummy" {
		t.Fatalf("unexpected payload content: %v", payload)
	}
}
