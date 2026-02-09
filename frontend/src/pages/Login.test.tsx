import { describe, it, expect, beforeEach, vi } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { render } from '../test/test-utils';
import Login from './Login';

const mockNavigate = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

vi.mock('../services/auth', () => ({
  login: vi.fn().mockResolvedValue({ token: 'mock-token', userId: 'mock-user-id' }),
  register: vi.fn().mockResolvedValue({ token: 'mock-token', userId: 'mock-user-id' }),
}));

describe('Login', () => {
  beforeEach(() => {
    sessionStorage.clear();
    mockNavigate.mockClear();
    vi.clearAllMocks();
  });

  it('should render login form', () => {
    render(<Login />);

    expect(screen.getByText('Samvaad')).toBeInTheDocument();
    expect(screen.getByText('Sign in to continue')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Username')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Password')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Sign In' })).toBeInTheDocument();
  });

  it('should show error when submitting empty username', async () => {
    const user = userEvent.setup();
    render(<Login />);

    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    expect(screen.getByText('Please enter a username')).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('should show error when username is too short', async () => {
    const user = userEvent.setup();
    render(<Login />);

    await user.type(screen.getByPlaceholderText('Username'), 'ab');
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    expect(screen.getByText('Username must be at least 3 characters')).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('should show error when password is empty', async () => {
    const user = userEvent.setup();
    render(<Login />);

    await user.type(screen.getByPlaceholderText('Username'), 'testuser');
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    expect(screen.getByText('Please enter a password')).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('should show error when password is too short', async () => {
    const user = userEvent.setup();
    render(<Login />);

    await user.type(screen.getByPlaceholderText('Username'), 'testuser');
    await user.type(screen.getByPlaceholderText('Password'), 'abc');
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    expect(screen.getByText('Password must be at least 4 characters')).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('should login and navigate to home on valid submission', async () => {
    const user = userEvent.setup();
    render(<Login />);

    await user.type(screen.getByPlaceholderText('Username'), 'testuser');
    await user.type(screen.getByPlaceholderText('Password'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Sign In' }));

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/home');
    });
    expect(sessionStorage.getItem('samvaad_user')).not.toBeNull();
  });

  it('should toggle to register mode', async () => {
    const user = userEvent.setup();
    render(<Login />);

    await user.click(screen.getByRole('button', { name: 'Register' }));

    expect(screen.getByText('Create an account')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Create Account' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Sign In' })).toBeInTheDocument();
  });

  it('should register and navigate to home on valid submission', async () => {
    const user = userEvent.setup();
    render(<Login />);

    await user.click(screen.getByRole('button', { name: 'Register' }));
    await user.type(screen.getByPlaceholderText('Username'), 'newuser');
    await user.type(screen.getByPlaceholderText('Password'), 'password123');
    await user.click(screen.getByRole('button', { name: 'Create Account' }));

    await waitFor(() => {
      expect(mockNavigate).toHaveBeenCalledWith('/home');
    });
  });

  it('should have autofocus on username input', () => {
    render(<Login />);
    expect(screen.getByPlaceholderText('Username')).toHaveFocus();
  });
});
