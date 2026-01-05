import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { wsService } from './websocket';

describe('WebSocketService', () => {
  let mockWsInstance: {
    readyState: number;
    onopen: (() => void) | null;
    onclose: (() => void) | null;
    onmessage: ((event: { data: string }) => void) | null;
    onerror: ((error: unknown) => void) | null;
    send: ReturnType<typeof vi.fn>;
    close: ReturnType<typeof vi.fn>;
  };

  beforeEach(() => {
    // Create fresh mock for each test
    mockWsInstance = {
      readyState: 1, // OPEN
      onopen: null,
      onclose: null,
      onmessage: null,
      onerror: null,
      send: vi.fn(),
      close: vi.fn().mockImplementation(function(this: typeof mockWsInstance) {
        this.readyState = 3; // CLOSED
        if (this.onclose) this.onclose();
      }),
    };

    globalThis.WebSocket = vi.fn().mockImplementation(() => mockWsInstance) as unknown as typeof WebSocket;
    (globalThis.WebSocket as unknown as { OPEN: number }).OPEN = 1;
  });

  afterEach(() => {
    wsService.disconnect();
    vi.clearAllMocks();
  });

  it('should connect to WebSocket server', async () => {
    const connectPromise = wsService.connect('ws://localhost:8080/ws');

    // Simulate successful connection
    mockWsInstance.onopen?.();

    await expect(connectPromise).resolves.toBeUndefined();
    expect(globalThis.WebSocket).toHaveBeenCalledWith('ws://localhost:8080/ws');
  });

  it('should reject on connection error', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    const connectPromise = wsService.connect('ws://localhost:8080/ws');

    // Simulate error
    mockWsInstance.onerror?.(new Error('Connection failed'));

    await expect(connectPromise).rejects.toBeDefined();
    consoleSpy.mockRestore();
  });

  it('should send message when connected', async () => {
    const connectPromise = wsService.connect('ws://localhost:8080/ws');
    mockWsInstance.onopen?.();
    await connectPromise;

    const message = { type: 'join', roomId: 'test-room' };
    wsService.send(message);

    expect(mockWsInstance.send).toHaveBeenCalledWith(JSON.stringify(message));
  });

  it('should not send message when disconnected', () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    const message = { type: 'join', roomId: 'test-room' };
    wsService.send(message);

    expect(consoleSpy).toHaveBeenCalledWith('WebSocket is not connected');
    consoleSpy.mockRestore();
  });

  it('should handle incoming messages', async () => {
    const connectPromise = wsService.connect('ws://localhost:8080/ws');
    mockWsInstance.onopen?.();
    await connectPromise;

    const messageHandler = vi.fn();
    wsService.onMessage(messageHandler);

    const testMessage = { type: 'user-joined', payload: { userId: 'test-user' } };
    mockWsInstance.onmessage?.({ data: JSON.stringify(testMessage) });

    expect(messageHandler).toHaveBeenCalledWith(testMessage);
  });

  it('should handle invalid JSON in messages', async () => {
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    const connectPromise = wsService.connect('ws://localhost:8080/ws');
    mockWsInstance.onopen?.();
    await connectPromise;

    const messageHandler = vi.fn();
    wsService.onMessage(messageHandler);

    mockWsInstance.onmessage?.({ data: 'invalid json' });

    expect(messageHandler).not.toHaveBeenCalled();
    expect(consoleSpy).toHaveBeenCalled();
    consoleSpy.mockRestore();
  });

  it('should unsubscribe message handler', async () => {
    const connectPromise = wsService.connect('ws://localhost:8080/ws');
    mockWsInstance.onopen?.();
    await connectPromise;

    const messageHandler = vi.fn();
    const unsubscribe = wsService.onMessage(messageHandler);

    // Unsubscribe
    unsubscribe();

    const testMessage = { type: 'test' };
    mockWsInstance.onmessage?.({ data: JSON.stringify(testMessage) });

    expect(messageHandler).not.toHaveBeenCalled();
  });

  it('should disconnect properly', async () => {
    const connectPromise = wsService.connect('ws://localhost:8080/ws');
    mockWsInstance.onopen?.();
    await connectPromise;

    wsService.disconnect();

    expect(mockWsInstance.close).toHaveBeenCalled();
  });

  it('should report connection status correctly', async () => {
    expect(wsService.isConnected).toBe(false);

    const connectPromise = wsService.connect('ws://localhost:8080/ws');
    mockWsInstance.onopen?.();
    await connectPromise;

    expect(wsService.isConnected).toBe(true);

    wsService.disconnect();

    expect(wsService.isConnected).toBe(false);
  });

  it('should set ws to null on close', async () => {
    const connectPromise = wsService.connect('ws://localhost:8080/ws');
    mockWsInstance.onopen?.();
    await connectPromise;

    expect(wsService.isConnected).toBe(true);

    // Simulate WebSocket close event
    mockWsInstance.onclose?.();

    expect(wsService.isConnected).toBe(false);
  });

  it('should handle disconnect when not connected', () => {
    // Should not throw
    expect(() => wsService.disconnect()).not.toThrow();
  });
});
