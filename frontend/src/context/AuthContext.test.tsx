import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { AuthProvider, useAuth } from './AuthContext';

// Mock the auth service
vi.mock('../services/auth', () => ({
  login: vi.fn().mockResolvedValue({ token: 'mock-token', userId: 'mock-user-id' }),
  register: vi.fn().mockResolvedValue({ token: 'mock-token', userId: 'mock-user-id' }),
}));

describe('AuthContext', () => {
  beforeEach(() => {
    sessionStorage.clear();
    vi.clearAllMocks();
  });

  it('should start with no user when sessionStorage is empty', () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: AuthProvider,
    });

    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });

  it('should login user and store in sessionStorage', async () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: AuthProvider,
    });

    await act(async () => {
      await result.current.login('testuser', 'password123');
    });

    await waitFor(() => {
      expect(result.current.user).not.toBeNull();
    });

    expect(result.current.user?.username).toBe('testuser');
    expect(result.current.user?.token).toBe('mock-token');
    expect(result.current.isAuthenticated).toBe(true);

    const stored = sessionStorage.getItem('samvaad_user');
    expect(stored).not.toBeNull();
    expect(JSON.parse(stored!).username).toBe('testuser');
  });

  it('should register user and store in sessionStorage', async () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: AuthProvider,
    });

    await act(async () => {
      await result.current.register('newuser', 'password123');
    });

    await waitFor(() => {
      expect(result.current.user).not.toBeNull();
    });

    expect(result.current.user?.username).toBe('newuser');
    expect(result.current.user?.token).toBe('mock-token');
    expect(result.current.isAuthenticated).toBe(true);
  });

  it('should logout user and clear sessionStorage', async () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: AuthProvider,
    });

    await act(async () => {
      await result.current.login('testuser', 'password123');
    });

    expect(result.current.isAuthenticated).toBe(true);

    act(() => {
      result.current.logout();
    });

    expect(result.current.user).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
    expect(sessionStorage.getItem('samvaad_user')).toBeNull();
  });

  it('should restore user from sessionStorage on mount', () => {
    const storedUser = { id: 'stored-id', username: 'storeduser', token: 'stored-token' };
    sessionStorage.setItem('samvaad_user', JSON.stringify(storedUser));

    const { result } = renderHook(() => useAuth(), {
      wrapper: AuthProvider,
    });

    expect(result.current.user).toEqual(storedUser);
    expect(result.current.isAuthenticated).toBe(true);
  });

  it('should throw error when useAuth is used outside AuthProvider', () => {
    expect(() => {
      renderHook(() => useAuth());
    }).toThrow('useAuth must be used within an AuthProvider');
  });
});
