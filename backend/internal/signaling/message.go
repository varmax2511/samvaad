package signaling

import "encoding/json"

type MessageType string

const (
	// room events
	MessageTypeJoin       MessageType = "join"
	MessageTypeLeave      MessageType = "leave"
	MessageTypeUserJoined MessageType = "user-joined"
	MessageTypeUserLeft   MessageType = "user-left"

	// WebRTC
	MessageTypeOffer        MessageType = "offer"
	MessageTypeAnswer       MessageType = "answer"
	MessageTypeIceCandidate MessageType = "ice-candidate"

	// chat
	MessageTypeChat MessageType = "chat"

	// errors
	MessageTypeError MessageType = "error"
)

// message structure
type Message struct {
	Type    MessageType     `json:"type"`
	RoomID  string          `json:"roomid,omitempty"`
	From    string          `json:"from,omitempty"`
	To      string          `json:"to,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type ChatPayload struct {
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

type JoinPayload struct {
	RoomID string `json:"roomId"`
	UserID string `json:"userId"`
}

type SDPPayload struct {
	SDP interface{} `json:"sdp"`
}

type ICEPayload struct {
	Candidate interface{} `json:"candidate"`
}
