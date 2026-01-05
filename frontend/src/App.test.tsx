import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import App from './App';

// Mock the pages to avoid complex setup
vi.mock('./pages/Login', () => ({
  default: () => <div data-testid="login-page">Login Page</div>,
}));

vi.mock('./pages/Home', () => ({
  default: () => <div data-testid="home-page">Home Page</div>,
}));

vi.mock('./pages/Call', () => ({
  default: () => <div data-testid="call-page">Call Page</div>,
}));

describe('App', () => {
  beforeEach(() => {
    sessionStorage.clear();
    window.history.pushState({}, '', '/');
  });

  it('should render login page when not authenticated', () => {
    render(<App />);
    expect(screen.getByTestId('login-page')).toBeInTheDocument();
  });

  it('should redirect to home when authenticated and on login page', async () => {
    sessionStorage.setItem('samvaad_user', JSON.stringify({ id: 'test-id', name: 'Test User' }));

    render(<App />);

    await waitFor(() => {
      expect(screen.getByTestId('home-page')).toBeInTheDocument();
    });
  });

  it('should render home page when authenticated and navigating to /home', async () => {
    sessionStorage.setItem('samvaad_user', JSON.stringify({ id: 'test-id', name: 'Test User' }));
    window.history.pushState({}, '', '/home');

    render(<App />);

    await waitFor(() => {
      expect(screen.getByTestId('home-page')).toBeInTheDocument();
    });
  });

  it('should render call page when authenticated and navigating to /call/:roomId', async () => {
    sessionStorage.setItem('samvaad_user', JSON.stringify({ id: 'test-id', name: 'Test User' }));
    window.history.pushState({}, '', '/call/TEST123');

    render(<App />);

    await waitFor(() => {
      expect(screen.getByTestId('call-page')).toBeInTheDocument();
    });
  });

  it('should redirect to login for unknown routes when not authenticated', () => {
    window.history.pushState({}, '', '/unknown-route');

    render(<App />);

    expect(screen.getByTestId('login-page')).toBeInTheDocument();
  });

  it('should redirect to home for unknown routes when authenticated', async () => {
    sessionStorage.setItem('samvaad_user', JSON.stringify({ id: 'test-id', name: 'Test User' }));
    window.history.pushState({}, '', '/unknown-route');

    render(<App />);

    await waitFor(() => {
      expect(screen.getByTestId('home-page')).toBeInTheDocument();
    });
  });

  it('should redirect to login when accessing protected routes without auth', () => {
    window.history.pushState({}, '', '/home');

    render(<App />);

    expect(screen.getByTestId('login-page')).toBeInTheDocument();
  });
});
