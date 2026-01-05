import { describe, it, expect, beforeEach, vi } from 'vitest';
import { screen } from '@testing-library/react';
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

describe('Login', () => {
  beforeEach(() => {
    sessionStorage.clear();
    mockNavigate.mockClear();
  });

  it('should render login form', () => {
    render(<Login />);

    expect(screen.getByText('Samvaad')).toBeInTheDocument();
    expect(screen.getByText('Simple video conversations')).toBeInTheDocument();
    expect(screen.getByPlaceholderText('Enter your name')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Continue' })).toBeInTheDocument();
  });

  it('should show error when submitting empty name', async () => {
    const user = userEvent.setup();
    render(<Login />);

    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(screen.getByText('Please enter your name')).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('should show error when name is too short', async () => {
    const user = userEvent.setup();
    render(<Login />);

    await user.type(screen.getByPlaceholderText('Enter your name'), 'A');
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(screen.getByText('Name must be at least 2 characters')).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('should show error when name is only whitespace', async () => {
    const user = userEvent.setup();
    render(<Login />);

    await user.type(screen.getByPlaceholderText('Enter your name'), '   ');
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(screen.getByText('Please enter your name')).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('should login and navigate to home on valid submission', async () => {
    const user = userEvent.setup();
    render(<Login />);

    await user.type(screen.getByPlaceholderText('Enter your name'), 'John Doe');
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(mockNavigate).toHaveBeenCalledWith('/home');
    expect(sessionStorage.getItem('samvaad_user')).not.toBeNull();
  });

  it('should clear error when typing after error', async () => {
    const user = userEvent.setup();
    render(<Login />);

    await user.click(screen.getByRole('button', { name: 'Continue' }));
    expect(screen.getByText('Please enter your name')).toBeInTheDocument();

    await user.type(screen.getByPlaceholderText('Enter your name'), 'John');
    await user.click(screen.getByRole('button', { name: 'Continue' }));

    expect(screen.queryByText('Please enter your name')).not.toBeInTheDocument();
  });

  it('should have autofocus on name input', () => {
    render(<Login />);
    expect(screen.getByPlaceholderText('Enter your name')).toHaveFocus();
  });
});
