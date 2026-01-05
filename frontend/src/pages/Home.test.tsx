import { describe, it, expect, beforeEach, vi } from 'vitest';
import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { render } from '../test/test-utils';
import Home from './Home';

const mockNavigate = vi.fn();

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

describe('Home', () => {
  beforeEach(() => {
    sessionStorage.clear();
    mockNavigate.mockClear();
    // Set up authenticated user
    sessionStorage.setItem('samvaad_user', JSON.stringify({ id: 'test-id', name: 'Test User' }));
  });

  it('should render home page with user name', () => {
    render(<Home />);

    expect(screen.getByText('Test User')).toBeInTheDocument();
    expect(screen.getByText('Start a conversation')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'New Call' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Log out' })).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Enter room code')).toBeInTheDocument();
  });

  it('should navigate to call with new room id when clicking New Call', async () => {
    const user = userEvent.setup();

    // Mock Math.random for predictable room id
    const mockRandom = vi.spyOn(Math, 'random').mockReturnValue(0.123456789);

    render(<Home />);

    await user.click(screen.getByRole('button', { name: 'New Call' }));

    expect(mockNavigate).toHaveBeenCalledWith(expect.stringMatching(/^\/call\/[A-Z0-9]+$/));

    mockRandom.mockRestore();
  });

  it('should show error when joining with empty room code', async () => {
    const user = userEvent.setup();
    render(<Home />);

    await user.click(screen.getByRole('button', { name: 'Join' }));

    expect(screen.getByText('Please enter a room code')).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('should navigate to call room when joining with valid code', async () => {
    const user = userEvent.setup();
    render(<Home />);

    await user.type(screen.getByPlaceholderText('Enter room code'), 'ABC123');
    await user.click(screen.getByRole('button', { name: 'Join' }));

    expect(mockNavigate).toHaveBeenCalledWith('/call/ABC123');
  });

  it('should convert room code to uppercase', async () => {
    const user = userEvent.setup();
    render(<Home />);

    const input = screen.getByPlaceholderText('Enter room code');
    await user.type(input, 'abc123');

    expect(input).toHaveValue('ABC123');
  });

  it('should trim room code before joining', async () => {
    const user = userEvent.setup();
    render(<Home />);

    await user.type(screen.getByPlaceholderText('Enter room code'), '  XYZ789  ');
    await user.click(screen.getByRole('button', { name: 'Join' }));

    expect(mockNavigate).toHaveBeenCalledWith('/call/XYZ789');
  });

  it('should logout and navigate to login when clicking Log out', async () => {
    const user = userEvent.setup();
    render(<Home />);

    await user.click(screen.getByRole('button', { name: 'Log out' }));

    expect(mockNavigate).toHaveBeenCalledWith('/');
    expect(sessionStorage.getItem('samvaad_user')).toBeNull();
  });

  it('should limit room code input to 10 characters', () => {
    render(<Home />);
    const input = screen.getByPlaceholderText('Enter room code');
    expect(input).toHaveAttribute('maxLength', '10');
  });
});
