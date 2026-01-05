import { describe, it, expect, beforeEach, vi } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { AuthProvider, useAuth } from './AuthContext';

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

  it('should login user and store in sessionStorage', () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: AuthProvider,
    });

    act(() => {
      result.current.login('Test User');
    });

    expect(result.current.user).not.toBeNull();
    expect(result.current.user?.name).toBe('Test User');
    expect(result.current.user?.id).toBe('test-uuid-1234');
    expect(result.current.isAuthenticated).toBe(true);

    const stored = sessionStorage.getItem('samvaad_user');
    expect(stored).not.toBeNull();
    expect(JSON.parse(stored!).name).toBe('Test User');
  });

  it('should trim whitespace from name on login', () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: AuthProvider,
    });

    act(() => {
      result.current.login('  Trimmed Name  ');
    });

    expect(result.current.user?.name).toBe('Trimmed Name');
  });

  it('should logout user and clear sessionStorage', () => {
    const { result } = renderHook(() => useAuth(), {
      wrapper: AuthProvider,
    });

    act(() => {
      result.current.login('Test User');
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
    const storedUser = { id: 'stored-id', name: 'Stored User' };
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
