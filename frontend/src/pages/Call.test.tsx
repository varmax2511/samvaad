import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithMemoryRouter } from '../test/test-utils';
import Call from './Call';
import { wsService } from '../services/websocket';

const mockNavigate = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ roomId: 'TEST123' }),
  };
});

vi.mock('../services/websocket', () => ({
  wsService: {
    connect: vi.fn().mockResolvedValue(undefined),
    disconnect: vi.fn(),
    send: vi.fn(),
    onMessage: vi.fn().mockReturnValue(() => true),
    isConnected: true,
  },
}));

vi.mock('../services/webrtc', () => ({
  WebRTCService: vi.fn().mockImplementation(() => ({
    initialize: vi.fn().mockResolvedValue({
      getTracks: () => [],
      getAudioTracks: () => [{ enabled: true }],
      getVideoTracks: () => [{ enabled: true }],
    }),
    createPeerConnection: vi.fn(),
    createOffer: vi.fn().mockResolvedValue({ type: 'offer', sdp: 'mock-sdp' }),
    createAnswer: vi.fn().mockResolvedValue({ type: 'answer', sdp: 'mock-sdp' }),
    handleAnswer: vi.fn().mockResolvedValue(undefined),
    addIceCandidate: vi.fn().mockResolvedValue(undefined),
    toggleAudio: vi.fn(),
    toggleVideo: vi.fn(),
    cleanup: vi.fn(),
    onRemoteStream: null,
    onIceCandidate: null,
    onConnectionStateChange: null,
  })),
}));

describe('Call', () => {
  beforeEach(() => {
    sessionStorage.clear();
    mockNavigate.mockClear();
    vi.clearAllMocks();

    // Set up authenticated user
    sessionStorage.setItem('samvaad_user', JSON.stringify({ id: 'test-id', username: 'testuser', token: 'test-token' }));

    // Reset the mock to default behavior
    vi.mocked(wsService.onMessage).mockReturnValue(() => true);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it('should redirect to home if no user', async () => {
    sessionStorage.clear();

    renderWithMemoryRouter(<Call />, { initialEntries: ['/call/TEST123'] });

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/');
    });
  });

  it('should render call interface', async () => {
    renderWithMemoryRouter(<Call />, { initialEntries: ['/call/TEST123'] });

    await waitFor(() => {
      expect(screen.getByText(/Waiting for others to join/i)).toBeInTheDocument();
      expect(screen.getByText(/Room: TEST123/i)).toBeInTheDocument();
    });
  });

  it('should connect to websocket and join room on mount', async () => {
    renderWithMemoryRouter(<Call />, { initialEntries: ['/call/TEST123'] });

    await waitFor(() => {
      expect(wsService.connect).toHaveBeenCalled();
      expect(wsService.send).toHaveBeenCalledWith({ type: 'join', roomId: 'TEST123' });
    });
  });

  it('should toggle audio when mute button is clicked', async () => {
    const user = userEvent.setup();
    renderWithMemoryRouter(<Call />, { initialEntries: ['/call/TEST123'] });

    await waitFor(() => {
      expect(screen.getByTitle('Mute')).toBeInTheDocument();
    });

    await user.click(screen.getByTitle('Mute'));

    expect(screen.getByTitle('Unmute')).toBeInTheDocument();

    await user.click(screen.getByTitle('Unmute'));

    expect(screen.getByTitle('Mute')).toBeInTheDocument();
  });

  it('should toggle video when video button is clicked', async () => {
    const user = userEvent.setup();
    renderWithMemoryRouter(<Call />, { initialEntries: ['/call/TEST123'] });

    await waitFor(() => {
      expect(screen.getByTitle('Stop Video')).toBeInTheDocument();
    });

    await user.click(screen.getByTitle('Stop Video'));

    expect(screen.getByTitle('Start Video')).toBeInTheDocument();

    await user.click(screen.getByTitle('Start Video'));

    expect(screen.getByTitle('Stop Video')).toBeInTheDocument();
  });

  it('should leave call when leave button is clicked', async () => {
    const user = userEvent.setup();
    renderWithMemoryRouter(<Call />, { initialEntries: ['/call/TEST123'] });

    await waitFor(() => {
      expect(screen.getByTitle('Leave Call')).toBeInTheDocument();
    });

    await user.click(screen.getByTitle('Leave Call'));

    expect(wsService.send).toHaveBeenCalledWith({ type: 'leave' });
    expect(wsService.disconnect).toHaveBeenCalled();
    expect(mockNavigate).toHaveBeenCalledWith('/home');
  });

  it('should display error message when error occurs', async () => {
    const errorMessage = 'Room is full';
    vi.mocked(wsService.onMessage).mockImplementation((handler) => {
      setTimeout(() => {
        handler({ type: 'error', payload: { message: errorMessage } });
      }, 10);
      return () => true;
    });

    renderWithMemoryRouter(<Call />, { initialEntries: ['/call/TEST123'] });

    await waitFor(() => {
      expect(screen.getByText(errorMessage)).toBeInTheDocument();
    });

    expect(screen.getByRole('button', { name: 'Go Back' })).toBeInTheDocument();
  });

  it('should navigate back to home from error screen', async () => {
    const user = userEvent.setup();
    vi.mocked(wsService.onMessage).mockImplementation((handler) => {
      setTimeout(() => {
        handler({ type: 'error', payload: { message: 'Test error' } });
      }, 10);
      return () => true;
    });

    renderWithMemoryRouter(<Call />, { initialEntries: ['/call/TEST123'] });

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Go Back' })).toBeInTheDocument();
    });

    await user.click(screen.getByRole('button', { name: 'Go Back' }));

    expect(mockNavigate).toHaveBeenCalledWith('/home');
  });

  it('should handle user-left message', async () => {
    vi.mocked(wsService.onMessage).mockImplementation((handler) => {
      setTimeout(() => {
        handler({
          type: 'user-left',
          payload: { userId: 'peer-id' },
        });
      }, 10);
      return () => true;
    });

    renderWithMemoryRouter(<Call />, { initialEntries: ['/call/TEST123'] });

    await waitFor(() => {
      expect(screen.getByText(/Waiting for others to join/i)).toBeInTheDocument();
    });
  });
});
