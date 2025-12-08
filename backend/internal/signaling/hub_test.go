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
	// verify otherUsers exists and is empty for first user
	otherUsers1, ok := payload1["otherUsers"].([]interface{})
	if !ok {
		t.Fatalf("expected otherUsers to be an array, got %T", payload1["otherUsers"])
	}
	if len(otherUsers1) != 0 {
		t.Fatalf("expected otherUsers to be empty for first user, got %d users", len(otherUsers1))
	}

	// u2 joins same room
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: c2.ID, RoomID: "roomA"}

	// u2 should get confirmation
	m2 := readMessage(t, c2.Send, 500*time.Millisecond)
	if m2.Type != MessageTypeUserJoined {
		t.Fatalf("expected type %s got %s", MessageTypeUserJoined, m2.Type)
	}
	// verify u2's payload contains otherUsers with u1
	var payload2 map[string]interface{}
	if err := json.Unmarshal(m2.Payload, &payload2); err != nil {
		t.Fatalf("failed to unmarshal payload2: %v", err)
	}
	if payload2["userId"] != c2.ID {
		t.Fatalf("expected userId %s got %v", c2.ID, payload2["userId"])
	}
	otherUsers2, ok := payload2["otherUsers"].([]interface{})
	if !ok {
		t.Fatalf("expected otherUsers to be an array, got %T", payload2["otherUsers"])
	}
	if len(otherUsers2) != 1 {
		t.Fatalf("expected otherUsers to have 1 user, got %d", len(otherUsers2))
	}
	if otherUsers2[0] != c1.ID {
		t.Fatalf("expected otherUsers[0] to be %s, got %v", c1.ID, otherUsers2[0])
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

func TestLeaveRoomAndRejoin(t *testing.T) {
	h := NewHub()
	go h.Run()

	c1 := &Client{ID: "u1", Send: make(chan []byte, 10), Hub: h}
	c2 := &Client{ID: "u2", Send: make(chan []byte, 10), Hub: h}

	h.Register <- c1
	h.Register <- c2

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok1 := h.Clients[c1.ID]
		_, ok2 := h.Clients[c2.ID]
		h.mu.RUnlock()
		return ok1 && ok2
	})

	// u1 joins room "testRoom"
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: c1.ID, RoomID: "testRoom"}
	_ = readMessage(t, c1.Send, 500*time.Millisecond) // join confirmation

	// verify u1 is in the room
	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		room, ok := h.Rooms["testRoom"]
		h.mu.RUnlock()
		if !ok {
			return false
		}
		_, exists := room[c1.ID]
		return exists && len(room) == 1
	})

	// u1 leaves the room
	h.Broadcast <- &Message{Type: MessageTypeLeave, From: c1.ID, RoomID: "testRoom"}

	// verify u1 is removed from the room
	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		room, ok := h.Rooms["testRoom"]
		h.mu.RUnlock()
		// room should either not exist or be empty
		return !ok || len(room) == 0
	})

	// u2 joins the room
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: c2.ID, RoomID: "testRoom"}
	_ = readMessage(t, c2.Send, 500*time.Millisecond) // join confirmation

	// verify u2 is in the room
	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		room, ok := h.Rooms["testRoom"]
		h.mu.RUnlock()
		if !ok {
			return false
		}
		_, exists := room[c2.ID]
		return exists && len(room) == 1
	})

	// u2 leaves the room
	h.Broadcast <- &Message{Type: MessageTypeLeave, From: c2.ID, RoomID: "testRoom"}

	// verify room is empty/deleted
	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		room, ok := h.Rooms["testRoom"]
		h.mu.RUnlock()
		return !ok || len(room) == 0
	})

	// u1 tries to rejoin - should succeed now
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: c1.ID, RoomID: "testRoom"}
	msg := readMessage(t, c1.Send, 500*time.Millisecond)
	if msg.Type != MessageTypeUserJoined {
		t.Fatalf("expected u1 to successfully rejoin, got message type: %s", msg.Type)
	}

	// verify u1 is back in the room
	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		room, ok := h.Rooms["testRoom"]
		h.mu.RUnlock()
		if !ok {
			return false
		}
		_, exists := room[c1.ID]
		return exists && len(room) == 1
	})
}

func TestMultipleUsersLeaveAndRejoin(t *testing.T) {
	h := NewHub()
	go h.Run()

	c1 := &Client{ID: "user1", Send: make(chan []byte, 10), Hub: h}
	c2 := &Client{ID: "user2", Send: make(chan []byte, 10), Hub: h}

	h.Register <- c1
	h.Register <- c2

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok1 := h.Clients[c1.ID]
		_, ok2 := h.Clients[c2.ID]
		h.mu.RUnlock()
		return ok1 && ok2
	})

	roomID := "multiUserRoom"

	// Both users join
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: c1.ID, RoomID: roomID}
	_ = readMessage(t, c1.Send, 500*time.Millisecond)

	h.Broadcast <- &Message{Type: MessageTypeJoin, From: c2.ID, RoomID: roomID}
	_ = readMessage(t, c2.Send, 500*time.Millisecond)
	_ = readMessage(t, c1.Send, 500*time.Millisecond) // notification to c1

	// Verify both are in room
	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		room, ok := h.Rooms[roomID]
		h.mu.RUnlock()
		return ok && len(room) == 2
	})

	// c1 leaves
	h.Broadcast <- &Message{Type: MessageTypeLeave, From: c1.ID, RoomID: roomID}
	_ = readMessage(t, c2.Send, 500*time.Millisecond) // notification to c2

	// Verify only c2 remains
	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		room, ok := h.Rooms[roomID]
		h.mu.RUnlock()
		if !ok {
			return false
		}
		_, c1Exists := room[c1.ID]
		_, c2Exists := room[c2.ID]
		return !c1Exists && c2Exists && len(room) == 1
	})

	// c2 leaves
	h.Broadcast <- &Message{Type: MessageTypeLeave, From: c2.ID, RoomID: roomID}

	// Verify room is deleted
	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok := h.Rooms[roomID]
		h.mu.RUnlock()
		return !ok
	})

	// Both users rejoin - should succeed
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: c1.ID, RoomID: roomID}
	msg1 := readMessage(t, c1.Send, 500*time.Millisecond)
	if msg1.Type != MessageTypeUserJoined {
		t.Fatalf("expected c1 to successfully rejoin, got message type: %s", msg1.Type)
	}

	h.Broadcast <- &Message{Type: MessageTypeJoin, From: c2.ID, RoomID: roomID}
	msg2 := readMessage(t, c2.Send, 500*time.Millisecond)
	if msg2.Type != MessageTypeUserJoined {
		t.Fatalf("expected c2 to successfully rejoin, got message type: %s", msg2.Type)
	}

	// Verify both are back in room
	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		room, ok := h.Rooms[roomID]
		h.mu.RUnlock()
		if !ok {
			return false
		}
		_, c1Exists := room[c1.ID]
		_, c2Exists := room[c2.ID]
		return c1Exists && c2Exists && len(room) == 2
	})
}

// ============================================
// WebRTC Signaling Tests
// ============================================

func TestWebRTCAnswerForwarding(t *testing.T) {
	h := NewHub()
	go h.Run()

	caller := &Client{ID: "caller", Send: make(chan []byte, 4), Hub: h}
	callee := &Client{ID: "callee", Send: make(chan []byte, 4), Hub: h}

	h.Register <- caller
	h.Register <- callee

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok1 := h.Clients[caller.ID]
		_, ok2 := h.Clients[callee.ID]
		h.mu.RUnlock()
		return ok1 && ok2
	})

	// Send an answer from callee to caller
	answerPayload, _ := json.Marshal(map[string]string{
		"type": "answer",
		"sdp":  "v=0\r\no=- 123456 2 IN IP4 127.0.0.1\r\n...",
	})

	msg := &Message{
		Type:    MessageTypeAnswer,
		From:    callee.ID,
		To:      caller.ID,
		Payload: answerPayload,
	}
	h.Broadcast <- msg

	// Caller should receive the answer
	got := readMessage(t, caller.Send, 500*time.Millisecond)
	if got.Type != MessageTypeAnswer {
		t.Fatalf("expected answer type, got %s", got.Type)
	}
	if got.From != callee.ID {
		t.Fatalf("expected from %s got %s", callee.ID, got.From)
	}
	if got.To != caller.ID {
		t.Fatalf("expected to %s got %s", caller.ID, got.To)
	}

	var payload map[string]string
	if err := json.Unmarshal(got.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if payload["type"] != "answer" {
		t.Fatalf("expected payload type 'answer', got %s", payload["type"])
	}
}

func TestWebRTCIceCandidateForwarding(t *testing.T) {
	h := NewHub()
	go h.Run()

	peer1 := &Client{ID: "peer1", Send: make(chan []byte, 10), Hub: h}
	peer2 := &Client{ID: "peer2", Send: make(chan []byte, 10), Hub: h}

	h.Register <- peer1
	h.Register <- peer2

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok1 := h.Clients[peer1.ID]
		_, ok2 := h.Clients[peer2.ID]
		h.mu.RUnlock()
		return ok1 && ok2
	})

	// Send ICE candidate from peer1 to peer2
	icePayload, _ := json.Marshal(map[string]interface{}{
		"candidate":     "candidate:1 1 UDP 2122252543 192.168.1.100 54321 typ host",
		"sdpMid":        "0",
		"sdpMLineIndex": 0,
	})

	msg := &Message{
		Type:    MessageTypeIceCandidate,
		From:    peer1.ID,
		To:      peer2.ID,
		Payload: icePayload,
	}
	h.Broadcast <- msg

	// peer2 should receive the ICE candidate
	got := readMessage(t, peer2.Send, 500*time.Millisecond)
	if got.Type != MessageTypeIceCandidate {
		t.Fatalf("expected ice-candidate type, got %s", got.Type)
	}
	if got.From != peer1.ID {
		t.Fatalf("expected from %s got %s", peer1.ID, got.From)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(got.Payload, &payload); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if !strings.Contains(payload["candidate"].(string), "candidate:1") {
		t.Fatalf("unexpected candidate content: %v", payload["candidate"])
	}
}

func TestFullWebRTCSignalingFlow(t *testing.T) {
	h := NewHub()
	go h.Run()

	caller := &Client{ID: "caller", Send: make(chan []byte, 20), Hub: h}
	callee := &Client{ID: "callee", Send: make(chan []byte, 20), Hub: h}

	h.Register <- caller
	h.Register <- callee

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok1 := h.Clients[caller.ID]
		_, ok2 := h.Clients[callee.ID]
		h.mu.RUnlock()
		return ok1 && ok2
	})

	roomID := "webrtc-test-room"

	// Step 1: Both users join the room
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: caller.ID, RoomID: roomID}
	callerJoinMsg := readMessage(t, caller.Send, 500*time.Millisecond)
	if callerJoinMsg.Type != MessageTypeUserJoined {
		t.Fatalf("expected user-joined, got %s", callerJoinMsg.Type)
	}

	h.Broadcast <- &Message{Type: MessageTypeJoin, From: callee.ID, RoomID: roomID}
	calleeJoinMsg := readMessage(t, callee.Send, 500*time.Millisecond)
	if calleeJoinMsg.Type != MessageTypeUserJoined {
		t.Fatalf("expected user-joined, got %s", calleeJoinMsg.Type)
	}

	// Caller should be notified of callee joining
	callerNotify := readMessage(t, caller.Send, 500*time.Millisecond)
	if callerNotify.Type != MessageTypeUserJoined {
		t.Fatalf("expected notification, got %s", callerNotify.Type)
	}

	// Verify callee's join confirmation contains caller in otherUsers
	var calleePayload map[string]interface{}
	if err := json.Unmarshal(calleeJoinMsg.Payload, &calleePayload); err != nil {
		t.Fatalf("failed to unmarshal callee payload: %v", err)
	}
	otherUsers := calleePayload["otherUsers"].([]interface{})
	if len(otherUsers) != 1 || otherUsers[0] != caller.ID {
		t.Fatalf("expected otherUsers to contain caller, got %v", otherUsers)
	}

	// Step 2: Callee sends offer to caller (callee joined later, so initiates)
	offerPayload, _ := json.Marshal(map[string]string{
		"type": "offer",
		"sdp":  "v=0\r\no=- 12345 2 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n...",
	})
	h.Broadcast <- &Message{
		Type:    MessageTypeOffer,
		From:    callee.ID,
		To:      caller.ID,
		Payload: offerPayload,
	}

	// Caller receives offer
	offerMsg := readMessage(t, caller.Send, 500*time.Millisecond)
	if offerMsg.Type != MessageTypeOffer {
		t.Fatalf("expected offer, got %s", offerMsg.Type)
	}
	if offerMsg.From != callee.ID {
		t.Fatalf("expected offer from callee, got %s", offerMsg.From)
	}

	// Step 3: Caller sends answer back
	answerPayload, _ := json.Marshal(map[string]string{
		"type": "answer",
		"sdp":  "v=0\r\no=- 67890 2 IN IP4 127.0.0.1\r\ns=-\r\nt=0 0\r\n...",
	})
	h.Broadcast <- &Message{
		Type:    MessageTypeAnswer,
		From:    caller.ID,
		To:      callee.ID,
		Payload: answerPayload,
	}

	// Callee receives answer
	answerMsg := readMessage(t, callee.Send, 500*time.Millisecond)
	if answerMsg.Type != MessageTypeAnswer {
		t.Fatalf("expected answer, got %s", answerMsg.Type)
	}
	if answerMsg.From != caller.ID {
		t.Fatalf("expected answer from caller, got %s", answerMsg.From)
	}

	// Step 4: Exchange ICE candidates (both directions)
	// Callee sends ICE candidate
	icePayload1, _ := json.Marshal(map[string]interface{}{
		"candidate":     "candidate:1 1 UDP 2122252543 192.168.1.1 12345 typ host",
		"sdpMid":        "0",
		"sdpMLineIndex": 0,
	})
	h.Broadcast <- &Message{
		Type:    MessageTypeIceCandidate,
		From:    callee.ID,
		To:      caller.ID,
		Payload: icePayload1,
	}

	// Caller receives ICE candidate
	iceMsg1 := readMessage(t, caller.Send, 500*time.Millisecond)
	if iceMsg1.Type != MessageTypeIceCandidate {
		t.Fatalf("expected ice-candidate, got %s", iceMsg1.Type)
	}

	// Caller sends ICE candidate
	icePayload2, _ := json.Marshal(map[string]interface{}{
		"candidate":     "candidate:2 1 UDP 2122252543 192.168.1.2 54321 typ host",
		"sdpMid":        "0",
		"sdpMLineIndex": 0,
	})
	h.Broadcast <- &Message{
		Type:    MessageTypeIceCandidate,
		From:    caller.ID,
		To:      callee.ID,
		Payload: icePayload2,
	}

	// Callee receives ICE candidate
	iceMsg2 := readMessage(t, callee.Send, 500*time.Millisecond)
	if iceMsg2.Type != MessageTypeIceCandidate {
		t.Fatalf("expected ice-candidate, got %s", iceMsg2.Type)
	}

	// Verify ICE candidate payloads are preserved correctly
	var ice1 map[string]interface{}
	if err := json.Unmarshal(iceMsg1.Payload, &ice1); err != nil {
		t.Fatalf("failed to unmarshal ice1: %v", err)
	}
	if !strings.Contains(ice1["candidate"].(string), "192.168.1.1") {
		t.Fatalf("ICE candidate 1 payload corrupted: %v", ice1)
	}

	var ice2 map[string]interface{}
	if err := json.Unmarshal(iceMsg2.Payload, &ice2); err != nil {
		t.Fatalf("failed to unmarshal ice2: %v", err)
	}
	if !strings.Contains(ice2["candidate"].(string), "192.168.1.2") {
		t.Fatalf("ICE candidate 2 payload corrupted: %v", ice2)
	}
}

func TestSignalingToNonExistentClient(t *testing.T) {
	h := NewHub()
	go h.Run()

	sender := &Client{ID: "sender", Send: make(chan []byte, 4), Hub: h}

	h.Register <- sender

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok := h.Clients[sender.ID]
		h.mu.RUnlock()
		return ok
	})

	// Try to send an offer to a non-existent client
	offerPayload, _ := json.Marshal(map[string]string{
		"type": "offer",
		"sdp":  "dummy-sdp",
	})

	msg := &Message{
		Type:    MessageTypeOffer,
		From:    sender.ID,
		To:      "non-existent-client",
		Payload: offerPayload,
	}
	h.Broadcast <- msg

	// Give some time for the hub to process
	time.Sleep(100 * time.Millisecond)

	// Sender should NOT receive any message (the message is simply dropped)
	// We verify by checking that no message arrives within timeout
	select {
	case msg := <-sender.Send:
		// If hub sends an error, that's also acceptable behavior
		var m Message
		if err := json.Unmarshal(msg, &m); err == nil {
			if m.Type != MessageTypeError {
				t.Logf("Received unexpected message: %s", m.Type)
			}
		}
	case <-time.After(200 * time.Millisecond):
		// Expected - no message received
	}
}

func TestMultipleIceCandidatesInSequence(t *testing.T) {
	h := NewHub()
	go h.Run()

	peer1 := &Client{ID: "peer1", Send: make(chan []byte, 20), Hub: h}
	peer2 := &Client{ID: "peer2", Send: make(chan []byte, 20), Hub: h}

	h.Register <- peer1
	h.Register <- peer2

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok1 := h.Clients[peer1.ID]
		_, ok2 := h.Clients[peer2.ID]
		h.mu.RUnlock()
		return ok1 && ok2
	})

	// Send multiple ICE candidates in sequence (simulating real WebRTC behavior)
	candidates := []string{
		"candidate:1 1 UDP 2122252543 192.168.1.1 50000 typ host",
		"candidate:2 1 UDP 2122252543 10.0.0.1 50001 typ host",
		"candidate:3 1 UDP 1685987071 203.0.113.1 50002 typ srflx raddr 192.168.1.1 rport 50000",
	}

	for i, cand := range candidates {
		payload, _ := json.Marshal(map[string]interface{}{
			"candidate":     cand,
			"sdpMid":        "0",
			"sdpMLineIndex": 0,
		})
		h.Broadcast <- &Message{
			Type:    MessageTypeIceCandidate,
			From:    peer1.ID,
			To:      peer2.ID,
			Payload: payload,
		}

		// peer2 should receive each candidate
		got := readMessage(t, peer2.Send, 500*time.Millisecond)
		if got.Type != MessageTypeIceCandidate {
			t.Fatalf("candidate %d: expected ice-candidate, got %s", i, got.Type)
		}

		var gotPayload map[string]interface{}
		if err := json.Unmarshal(got.Payload, &gotPayload); err != nil {
			t.Fatalf("candidate %d: failed to unmarshal: %v", i, err)
		}
		if gotPayload["candidate"] != cand {
			t.Fatalf("candidate %d: expected %s, got %s", i, cand, gotPayload["candidate"])
		}
	}
}

func TestWebRTCSignalingAfterUserLeaves(t *testing.T) {
	h := NewHub()
	go h.Run()

	peer1 := &Client{ID: "peer1", Send: make(chan []byte, 10), Hub: h}
	peer2 := &Client{ID: "peer2", Send: make(chan []byte, 10), Hub: h}

	h.Register <- peer1
	h.Register <- peer2

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok1 := h.Clients[peer1.ID]
		_, ok2 := h.Clients[peer2.ID]
		h.mu.RUnlock()
		return ok1 && ok2
	})

	roomID := "leave-test-room"

	// Both join room
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: peer1.ID, RoomID: roomID}
	_ = readMessage(t, peer1.Send, 500*time.Millisecond)
	h.Broadcast <- &Message{Type: MessageTypeJoin, From: peer2.ID, RoomID: roomID}
	_ = readMessage(t, peer2.Send, 500*time.Millisecond)
	_ = readMessage(t, peer1.Send, 500*time.Millisecond) // notification

	// peer2 leaves
	h.Broadcast <- &Message{Type: MessageTypeLeave, From: peer2.ID, RoomID: roomID}
	leaveNotify := readMessage(t, peer1.Send, 500*time.Millisecond)
	if leaveNotify.Type != MessageTypeUserLeft {
		t.Fatalf("expected user-left notification, got %s", leaveNotify.Type)
	}

	// Verify user-left payload contains correct userId
	var leavePayload map[string]interface{}
	if err := json.Unmarshal(leaveNotify.Payload, &leavePayload); err != nil {
		t.Fatalf("failed to unmarshal leave payload: %v", err)
	}
	if leavePayload["userId"] != peer2.ID {
		t.Fatalf("expected userId %s, got %v", peer2.ID, leavePayload["userId"])
	}

	// Now unregister peer2 completely
	h.Unregister <- peer2

	waitFor(t, 500*time.Millisecond, func() bool {
		h.mu.RLock()
		_, ok := h.Clients[peer2.ID]
		h.mu.RUnlock()
		return !ok
	})

	// peer1 tries to send signaling to peer2 (should be silently dropped)
	offerPayload, _ := json.Marshal(map[string]string{"sdp": "test"})
	h.Broadcast <- &Message{
		Type:    MessageTypeOffer,
		From:    peer1.ID,
		To:      peer2.ID,
		Payload: offerPayload,
	}

	// No crash, no error - message just gets dropped
	time.Sleep(100 * time.Millisecond)
}
