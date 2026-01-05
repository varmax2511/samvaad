import { useEffect, useRef, useState, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { wsService } from '../services/websocket';
import { WebRTCService } from '../services/webrtc';
import type { Message, JoinPayload, UserPayload, RTCPayload } from '../types';
import './Call.css';

const WS_URL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws`;

export default function Call() {
  const { roomId } = useParams<{ roomId: string }>();
  const { user } = useAuth();
  const navigate = useNavigate();

  const localVideoRef = useRef<HTMLVideoElement>(null);
  const remoteVideoRef = useRef<HTMLVideoElement>(null);
  const webrtcRef = useRef<WebRTCService | null>(null);

  const [, setIsConnected] = useState(false);
  const [isAudioEnabled, setIsAudioEnabled] = useState(true);
  const [isVideoEnabled, setIsVideoEnabled] = useState(true);
  const [peerConnected, setPeerConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [peerId, setPeerId] = useState<string | null>(null);

  const handleLeave = useCallback(() => {
    wsService.send({ type: 'leave' });
    wsService.disconnect();
    webrtcRef.current?.cleanup();
    navigate('/home');
  }, [navigate]);

  useEffect(() => {
    if (!user || !roomId) {
      navigate('/');
      return;
    }

    let isMounted = true;
    const webrtc = new WebRTCService();
    webrtcRef.current = webrtc;

    const initializeCall = async () => {
      try {
        const localStream = await webrtc.initialize();
        if (!isMounted) return;

        if (localVideoRef.current) {
          localVideoRef.current.srcObject = localStream;
        }

        webrtc.onRemoteStream = (stream) => {
          if (remoteVideoRef.current) {
            remoteVideoRef.current.srcObject = stream;
          }
          setPeerConnected(true);
        };

        webrtc.onIceCandidate = (candidate) => {
          if (peerId) {
            wsService.send({
              type: 'ice-candidate',
              to: peerId,
              payload: { candidate: candidate.toJSON() },
            });
          }
        };

        webrtc.onConnectionStateChange = (state) => {
          if (state === 'disconnected' || state === 'failed') {
            setPeerConnected(false);
          }
        };

        await wsService.connect(WS_URL);
        if (!isMounted) return;

        setIsConnected(true);

        const unsubscribe = wsService.onMessage(async (message: Message) => {
          switch (message.type) {
            case 'user-joined': {
              const payload = message.payload as JoinPayload | UserPayload;
              if ('otherUsers' in payload) {
                if (payload.otherUsers.length > 0) {
                  const remotePeerId = payload.otherUsers[0];
                  setPeerId(remotePeerId);
                  webrtc.createPeerConnection();
                  const offer = await webrtc.createOffer();
                  wsService.send({
                    type: 'offer',
                    to: remotePeerId,
                    payload: { sdp: offer },
                  });
                }
              } else {
                setPeerId(payload.userId);
              }
              break;
            }

            case 'user-left': {
              setPeerConnected(false);
              setPeerId(null);
              if (remoteVideoRef.current) {
                remoteVideoRef.current.srcObject = null;
              }
              break;
            }

            case 'offer': {
              const { sdp } = message.payload as RTCPayload;
              if (sdp && message.from) {
                setPeerId(message.from);
                webrtc.createPeerConnection();
                webrtc.onIceCandidate = (candidate) => {
                  wsService.send({
                    type: 'ice-candidate',
                    to: message.from!,
                    payload: { candidate: candidate.toJSON() },
                  });
                };
                const answer = await webrtc.createAnswer(sdp);
                wsService.send({
                  type: 'answer',
                  to: message.from,
                  payload: { sdp: answer },
                });
              }
              break;
            }

            case 'answer': {
              const { sdp } = message.payload as RTCPayload;
              if (sdp) {
                await webrtc.handleAnswer(sdp);
              }
              break;
            }

            case 'ice-candidate': {
              const { candidate } = message.payload as RTCPayload;
              if (candidate) {
                await webrtc.addIceCandidate(candidate);
              }
              break;
            }

            case 'error': {
              const errorPayload = message.payload as { message: string };
              setError(errorPayload.message);
              break;
            }
          }
        });

        wsService.send({ type: 'join', roomId });

        return () => {
          unsubscribe();
        };
      } catch (err) {
        if (!isMounted) return;
        setError(err instanceof Error ? err.message : 'Failed to initialize call');
      }
    };

    initializeCall();

    return () => {
      isMounted = false;
      wsService.disconnect();
      webrtc.cleanup();
    };
  }, [user, roomId, navigate]);

  const toggleAudio = () => {
    const newState = !isAudioEnabled;
    setIsAudioEnabled(newState);
    webrtcRef.current?.toggleAudio(newState);
  };

  const toggleVideo = () => {
    const newState = !isVideoEnabled;
    setIsVideoEnabled(newState);
    webrtcRef.current?.toggleVideo(newState);
  };

  if (error) {
    return (
      <div className="call-error">
        <p>{error}</p>
        <button onClick={() => navigate('/home')}>Go Back</button>
      </div>
    );
  }

  return (
    <div className="call-container">
      <div className="video-area">
        <video
          ref={remoteVideoRef}
          className={`remote-video ${peerConnected ? 'active' : ''}`}
          autoPlay
          playsInline
        />

        {!peerConnected && (
          <div className="waiting-message">
            <p>Waiting for others to join...</p>
            <p className="room-code">Room: {roomId}</p>
          </div>
        )}

        <video
          ref={localVideoRef}
          className="local-video"
          autoPlay
          playsInline
          muted
        />
      </div>

      <div className="controls">
        <button
          onClick={toggleAudio}
          className={`control-button ${!isAudioEnabled ? 'off' : ''}`}
          title={isAudioEnabled ? 'Mute' : 'Unmute'}
        >
          {isAudioEnabled ? (
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z" />
              <path d="M19 10v2a7 7 0 0 1-14 0v-2" />
              <line x1="12" y1="19" x2="12" y2="23" />
              <line x1="8" y1="23" x2="16" y2="23" />
            </svg>
          ) : (
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <line x1="1" y1="1" x2="23" y2="23" />
              <path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6" />
              <path d="M17 16.95A7 7 0 0 1 5 12v-2m14 0v2a7 7 0 0 1-.11 1.23" />
              <line x1="12" y1="19" x2="12" y2="23" />
              <line x1="8" y1="23" x2="16" y2="23" />
            </svg>
          )}
        </button>

        <button
          onClick={toggleVideo}
          className={`control-button ${!isVideoEnabled ? 'off' : ''}`}
          title={isVideoEnabled ? 'Stop Video' : 'Start Video'}
        >
          {isVideoEnabled ? (
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <polygon points="23 7 16 12 23 17 23 7" />
              <rect x="1" y="5" width="15" height="14" rx="2" ry="2" />
            </svg>
          ) : (
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M16 16v1a2 2 0 0 1-2 2H3a2 2 0 0 1-2-2V7a2 2 0 0 1 2-2h2m5.66 0H14a2 2 0 0 1 2 2v3.34l1 1L23 7v10" />
              <line x1="1" y1="1" x2="23" y2="23" />
            </svg>
          )}
        </button>

        <button
          onClick={handleLeave}
          className="control-button leave"
          title="Leave Call"
        >
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M10.68 13.31a16 16 0 0 0 3.41 2.6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7 2 2 0 0 1 1.72 2v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.42 19.42 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.63A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91" />
            <line x1="23" y1="1" x2="1" y2="23" />
          </svg>
        </button>
      </div>
    </div>
  );
}
