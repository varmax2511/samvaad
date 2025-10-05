## About
A video calling app. 


## Phase1 Implementation Plan
Quick-Start Implementation Plan (Phase 1)
Week 1: Core Infrastructure
Days 1-2: Signaling Server

WebSocket server in Golang
Basic room join/leave logic
Message relay (SDP offer/answer, ICE candidates)

Days 3-4: Basic Frontend

WebRTC peer connection setup
Camera/microphone access
Basic UI (two video elements)

Days 5-7: Connection Flow

Integrate signaling with WebRTC
Test peer-to-peer connection
Handle connection failures

Week 2: Features
Days 8-10: Screen Sharing

Add screen capture API
Toggle between camera and screen
Remote side display logic

Days 11-12: Chat

WebRTC data channel for chat
Chat UI component
Message history (optional)

Days 13-14: Polish

Error handling
Connection quality indicators
Basic responsive design
Testing different network conditions

## Expected structure 
video-calling-app/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── signaling/
│   │   │   ├── hub.go          # WebSocket hub
│   │   │   ├── client.go       # Client connection
│   │   │   └── room.go         # Room management
│   │   ├── auth/
│   │   │   └── jwt.go
│   │   └── models/
│   │       └── user.go
│   ├── pkg/
│   │   └── config/
│   ├── go.mod
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── VideoCall.jsx
│   │   │   ├── Chat.jsx
│   │   │   └── Controls.jsx
│   │   ├── services/
│   │   │   ├── signaling.js
│   │   │   └── webrtc.js
│   │   └── App.jsx
│   ├── package.json
│   └── Dockerfile
├── docker-compose.yml
└── README.md