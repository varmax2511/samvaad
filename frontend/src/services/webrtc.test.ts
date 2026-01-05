import { describe, it, expect, beforeEach, vi } from 'vitest';
import { WebRTCService } from './webrtc';

describe('WebRTCService', () => {
  let webrtc: WebRTCService;
  let mockPeerConnection: {
    ontrack: ((event: unknown) => void) | null;
    onicecandidate: ((event: unknown) => void) | null;
    onconnectionstatechange: (() => void) | null;
    connectionState: string;
    addTrack: ReturnType<typeof vi.fn>;
    close: ReturnType<typeof vi.fn>;
    createOffer: ReturnType<typeof vi.fn>;
    createAnswer: ReturnType<typeof vi.fn>;
    setLocalDescription: ReturnType<typeof vi.fn>;
    setRemoteDescription: ReturnType<typeof vi.fn>;
    addIceCandidate: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    webrtc = new WebRTCService();

    mockPeerConnection = {
      ontrack: null,
      onicecandidate: null,
      onconnectionstatechange: null,
      connectionState: 'new',
      addTrack: vi.fn(),
      close: vi.fn(),
      createOffer: vi.fn().mockResolvedValue({ type: 'offer', sdp: 'mock-offer-sdp' }),
      createAnswer: vi.fn().mockResolvedValue({ type: 'answer', sdp: 'mock-answer-sdp' }),
      setLocalDescription: vi.fn().mockResolvedValue(undefined),
      setRemoteDescription: vi.fn().mockResolvedValue(undefined),
      addIceCandidate: vi.fn().mockResolvedValue(undefined),
    };

    globalThis.RTCPeerConnection = vi.fn().mockImplementation(() => mockPeerConnection) as unknown as typeof RTCPeerConnection;
  });

  describe('initialize', () => {
    it('should get user media and return local stream', async () => {
      const stream = await webrtc.initialize();

      expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalledWith({
        video: true,
        audio: true,
      });
      expect(stream).toBeDefined();
    });
  });

  describe('createPeerConnection', () => {
    it('should create a new RTCPeerConnection', () => {
      const pc = webrtc.createPeerConnection();

      expect(globalThis.RTCPeerConnection).toHaveBeenCalled();
      expect(pc).toBe(mockPeerConnection);
    });

    it('should add local tracks to peer connection when stream exists', async () => {
      await webrtc.initialize();
      webrtc.createPeerConnection();

      expect(mockPeerConnection.addTrack).toHaveBeenCalled();
    });

    it('should call onRemoteStream when track is received', async () => {
      const onRemoteStream = vi.fn();
      webrtc.onRemoteStream = onRemoteStream;

      webrtc.createPeerConnection();

      const mockTrack = { kind: 'video' };
      const mockStream = {
        getTracks: () => [mockTrack],
      };

      mockPeerConnection.ontrack!({
        streams: [mockStream],
      });

      expect(onRemoteStream).toHaveBeenCalled();
    });

    it('should call onIceCandidate when ICE candidate is received', () => {
      const onIceCandidate = vi.fn();
      webrtc.onIceCandidate = onIceCandidate;

      webrtc.createPeerConnection();

      const mockCandidate = { candidate: 'test-candidate' };
      mockPeerConnection.onicecandidate!({ candidate: mockCandidate });

      expect(onIceCandidate).toHaveBeenCalledWith(mockCandidate);
    });

    it('should not call onIceCandidate when candidate is null', () => {
      const onIceCandidate = vi.fn();
      webrtc.onIceCandidate = onIceCandidate;

      webrtc.createPeerConnection();

      mockPeerConnection.onicecandidate!({ candidate: null });

      expect(onIceCandidate).not.toHaveBeenCalled();
    });

    it('should call onConnectionStateChange when connection state changes', () => {
      const onConnectionStateChange = vi.fn();
      webrtc.onConnectionStateChange = onConnectionStateChange;

      webrtc.createPeerConnection();

      mockPeerConnection.connectionState = 'connected';
      mockPeerConnection.onconnectionstatechange!();

      expect(onConnectionStateChange).toHaveBeenCalledWith('connected');
    });
  });

  describe('createOffer', () => {
    it('should create and set local description', async () => {
      webrtc.createPeerConnection();
      const offer = await webrtc.createOffer();

      expect(mockPeerConnection.createOffer).toHaveBeenCalled();
      expect(mockPeerConnection.setLocalDescription).toHaveBeenCalledWith({
        type: 'offer',
        sdp: 'mock-offer-sdp',
      });
      expect(offer).toEqual({ type: 'offer', sdp: 'mock-offer-sdp' });
    });

    it('should create peer connection if not exists', async () => {
      const offer = await webrtc.createOffer();

      expect(globalThis.RTCPeerConnection).toHaveBeenCalled();
      expect(offer).toBeDefined();
    });
  });

  describe('createAnswer', () => {
    it('should set remote description and create answer', async () => {
      webrtc.createPeerConnection();

      const incomingOffer = { type: 'offer' as RTCSdpType, sdp: 'incoming-offer-sdp' };
      const answer = await webrtc.createAnswer(incomingOffer);

      expect(mockPeerConnection.setRemoteDescription).toHaveBeenCalled();
      expect(mockPeerConnection.createAnswer).toHaveBeenCalled();
      expect(mockPeerConnection.setLocalDescription).toHaveBeenCalledWith({
        type: 'answer',
        sdp: 'mock-answer-sdp',
      });
      expect(answer).toEqual({ type: 'answer', sdp: 'mock-answer-sdp' });
    });

    it('should create peer connection if not exists', async () => {
      const incomingOffer = { type: 'offer' as RTCSdpType, sdp: 'incoming-offer-sdp' };
      await webrtc.createAnswer(incomingOffer);

      expect(globalThis.RTCPeerConnection).toHaveBeenCalled();
    });
  });

  describe('handleAnswer', () => {
    it('should set remote description with answer', async () => {
      webrtc.createPeerConnection();

      const answer = { type: 'answer' as RTCSdpType, sdp: 'answer-sdp' };
      await webrtc.handleAnswer(answer);

      expect(mockPeerConnection.setRemoteDescription).toHaveBeenCalled();
    });

    it('should do nothing if peer connection does not exist', async () => {
      const answer = { type: 'answer' as RTCSdpType, sdp: 'answer-sdp' };
      await webrtc.handleAnswer(answer);

      expect(mockPeerConnection.setRemoteDescription).not.toHaveBeenCalled();
    });
  });

  describe('addIceCandidate', () => {
    it('should add ICE candidate to peer connection', async () => {
      webrtc.createPeerConnection();

      const candidate = { candidate: 'test-candidate', sdpMid: '0', sdpMLineIndex: 0 };
      await webrtc.addIceCandidate(candidate);

      expect(mockPeerConnection.addIceCandidate).toHaveBeenCalled();
    });

    it('should do nothing if peer connection does not exist', async () => {
      const candidate = { candidate: 'test-candidate', sdpMid: '0', sdpMLineIndex: 0 };
      await webrtc.addIceCandidate(candidate);

      expect(mockPeerConnection.addIceCandidate).not.toHaveBeenCalled();
    });
  });

  describe('toggleAudio', () => {
    it('should toggle audio track enabled state', async () => {
      const mockAudioTrack = { enabled: true };
      const mockStream = {
        getTracks: () => [mockAudioTrack],
        getAudioTracks: () => [mockAudioTrack],
        getVideoTracks: () => [],
        addTrack: vi.fn(),
      };

      vi.mocked(navigator.mediaDevices.getUserMedia).mockResolvedValue(mockStream as unknown as MediaStream);

      await webrtc.initialize();

      webrtc.toggleAudio(false);
      expect(mockAudioTrack.enabled).toBe(false);

      webrtc.toggleAudio(true);
      expect(mockAudioTrack.enabled).toBe(true);
    });
  });

  describe('toggleVideo', () => {
    it('should toggle video track enabled state', async () => {
      const mockVideoTrack = { enabled: true };
      const mockStream = {
        getTracks: () => [mockVideoTrack],
        getAudioTracks: () => [],
        getVideoTracks: () => [mockVideoTrack],
        addTrack: vi.fn(),
      };

      vi.mocked(navigator.mediaDevices.getUserMedia).mockResolvedValue(mockStream as unknown as MediaStream);

      await webrtc.initialize();

      webrtc.toggleVideo(false);
      expect(mockVideoTrack.enabled).toBe(false);

      webrtc.toggleVideo(true);
      expect(mockVideoTrack.enabled).toBe(true);
    });
  });

  describe('cleanup', () => {
    it('should stop all tracks and close peer connection', async () => {
      const mockTrack = { stop: vi.fn() };
      const mockStream = {
        getTracks: () => [mockTrack],
        getAudioTracks: () => [],
        getVideoTracks: () => [],
        addTrack: vi.fn(),
      };

      vi.mocked(navigator.mediaDevices.getUserMedia).mockResolvedValue(mockStream as unknown as MediaStream);

      await webrtc.initialize();
      webrtc.createPeerConnection();

      webrtc.cleanup();

      expect(mockTrack.stop).toHaveBeenCalled();
      expect(mockPeerConnection.close).toHaveBeenCalled();
    });

    it('should handle cleanup when no stream or connection exists', () => {
      expect(() => webrtc.cleanup()).not.toThrow();
    });
  });
});
