package signaling

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestClient_ReadMessages_Broadcasts(t *testing.T) {
	// setup server
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	hubCh := make(chan *Hub, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// upgrade
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// create hub with channels expected by client.go
		hub := &Hub{
			Broadcast:  make(chan *Message, 1),
			Unregister: make(chan *Client, 1),
		}
		// create client and run ReadMessages
		c := NewClient("test-client-id", hub, conn)
		go c.ReadMessages()
		// expose hub to test
		hubCh <- hub
		// keep handler alive until client unregisters or a short timeout
		select {
		case <-hub.Unregister:
		case <-time.After(2 * time.Second):
		}
	}))

	defer ts.Close()

	// dial server as websocket client
	url := "ws://" + strings.TrimPrefix(ts.URL, "http://") + "/"
	dialer := websocket.Dialer{}
	wsConn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer wsConn.Close()

	// get hub from server handler
	var hub *Hub
	select {
	case hub = <-hubCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for hub")
	}

	// send a JSON message; fields besides From are unknown, but server should set From to client ID
	raw := []byte(`{"type":"test","payload":"hello"}`)
	if err := wsConn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write message failed: %v", err)
	}

	// expect a message on hub.Broadcast with From == "test-client-id"
	select {
	case m := <-hub.Broadcast:
		if m == nil {
			t.Fatalf("received nil message")
		}
		if m.From != "test-client-id" {
			t.Fatalf("expected From=%q got=%q", "test-client-id", m.From)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast message")
	}
}

func TestClient_WriteMessages_Sends(t *testing.T) {
	// setup server
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	clientCh := make(chan *Client, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// upgrade
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// create hub (may be used by client)
		hub := &Hub{
			Broadcast:  make(chan *Message),
			Unregister: make(chan *Client),
		}
		// create client and run WriteMessages
		c := NewClient("writer-client", hub, conn)
		go c.WriteMessages()
		// expose client to test so it can send on Send channel
		clientCh <- c
		// keep handler alive until a short timeout
		time.Sleep(2 * time.Second)
	}))

	defer ts.Close()

	// dial server as websocket client
	url := "ws://" + strings.TrimPrefix(ts.URL, "http://") + "/"
	dialer := websocket.Dialer{}
	wsConn, _, err := dialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer wsConn.Close()

	// get client from server handler
	var c *Client
	select {
	case c = <-clientCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for client")
	}

	// send a message into client's Send channel
	msg := []byte("hello-from-server-client")
	select {
	case c.Send <- msg:
	case <-time.After(time.Second):
		t.Fatal("timed out sending to client.Send")
	}

	// read message on the websocket connection (should receive what client.WriteMessages wrote)
	wsConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, got, err := wsConn.ReadMessage()
	if err != nil {
		t.Fatalf("read message failed: %v", err)
	}
	if string(got) != string(msg) {
		t.Fatalf("expected message %q got %q", string(msg), string(got))
	}
}
