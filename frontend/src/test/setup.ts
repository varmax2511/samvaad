import '@testing-library/jest-dom';
import { vi } from 'vitest';

// Mock matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

// Mock crypto.randomUUID
Object.defineProperty(globalThis.crypto, 'randomUUID', {
  value: () => 'test-uuid-1234',
});

// Mock MediaStream
class MockMediaStream {
  private tracks: Array<{ kind: string; enabled: boolean; stop: () => void }> = [];

  constructor(tracks?: Array<{ kind: string; enabled: boolean; stop: () => void }>) {
    this.tracks = tracks || [];
  }

  getTracks() {
    return this.tracks;
  }

  getAudioTracks() {
    return this.tracks.filter((t) => t.kind === 'audio');
  }

  getVideoTracks() {
    return this.tracks.filter((t) => t.kind === 'video');
  }

  addTrack(track: { kind: string; enabled: boolean; stop: () => void }) {
    this.tracks.push(track);
  }
}

globalThis.MediaStream = MockMediaStream as unknown as typeof MediaStream;

// Mock MediaDevices
const mockAudioTrack = { kind: 'audio', enabled: true, stop: vi.fn() };
const mockVideoTrack = { kind: 'video', enabled: true, stop: vi.fn() };

const mockMediaStream = new MockMediaStream([mockAudioTrack, mockVideoTrack]);

Object.defineProperty(navigator, 'mediaDevices', {
  value: {
    getUserMedia: vi.fn().mockResolvedValue(mockMediaStream),
  },
  configurable: true,
});

// Mock RTCPeerConnection
class MockRTCPeerConnection {
  ontrack: ((event: unknown) => void) | null = null;
  onicecandidate: ((event: unknown) => void) | null = null;
  onconnectionstatechange: (() => void) | null = null;
  connectionState = 'new';
  localDescription: RTCSessionDescriptionInit | null = null;
  remoteDescription: RTCSessionDescriptionInit | null = null;

  addTrack = vi.fn();
  close = vi.fn();

  createOffer = vi.fn().mockResolvedValue({ type: 'offer', sdp: 'mock-offer-sdp' });
  createAnswer = vi.fn().mockResolvedValue({ type: 'answer', sdp: 'mock-answer-sdp' });

  setLocalDescription = vi.fn().mockImplementation((desc: RTCSessionDescriptionInit) => {
    this.localDescription = desc;
    return Promise.resolve();
  });

  setRemoteDescription = vi.fn().mockImplementation((desc: RTCSessionDescriptionInit) => {
    this.remoteDescription = desc;
    return Promise.resolve();
  });

  addIceCandidate = vi.fn().mockResolvedValue(undefined);
}

globalThis.RTCPeerConnection = MockRTCPeerConnection as unknown as typeof RTCPeerConnection;
globalThis.RTCSessionDescription = vi.fn().mockImplementation((init) => init) as unknown as typeof RTCSessionDescription;
globalThis.RTCIceCandidate = vi.fn().mockImplementation((init) => init) as unknown as typeof RTCIceCandidate;

// Mock WebSocket
class MockWebSocket {
  static CONNECTING = 0;
  static OPEN = 1;
  static CLOSING = 2;
  static CLOSED = 3;

  readyState = MockWebSocket.OPEN;
  onopen: (() => void) | null = null;
  onclose: (() => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  onerror: ((error: unknown) => void) | null = null;

  constructor(_url: string) {
    setTimeout(() => {
      if (this.onopen) this.onopen();
    }, 0);
  }

  send = vi.fn();
  close = vi.fn().mockImplementation(() => {
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) this.onclose();
  });
}

globalThis.WebSocket = MockWebSocket as unknown as typeof WebSocket;
