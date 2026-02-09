import { useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import './Login.css';

export default function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [isRegisterMode, setIsRegisterMode] = useState(false);
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const { login, register } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');

    if (!username.trim()) {
      setError('Please enter a username');
      return;
    }

    if (username.trim().length < 3) {
      setError('Username must be at least 3 characters');
      return;
    }

    if (!password) {
      setError('Please enter a password');
      return;
    }

    if (password.length < 4) {
      setError('Password must be at least 4 characters');
      return;
    }

    setIsLoading(true);

    try {
      if (isRegisterMode) {
        await register(username.trim(), password);
      } else {
        await login(username.trim(), password);
      }
      navigate('/home');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Authentication failed');
    } finally {
      setIsLoading(false);
    }
  };

  const toggleMode = () => {
    setIsRegisterMode(!isRegisterMode);
    setError('');
  };

  return (
    <div className="login-container">
      <div className="login-card">
        <h1 className="login-title">Samvaad</h1>
        <p className="login-subtitle">
          {isRegisterMode ? 'Create an account' : 'Sign in to continue'}
        </p>

        <form onSubmit={handleSubmit} className="login-form">
          <div className="input-group">
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="Username"
              className="login-input"
              autoFocus
              disabled={isLoading}
            />
          </div>

          <div className="input-group">
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Password"
              className="login-input"
              disabled={isLoading}
            />
          </div>

          {error && <p className="login-error">{error}</p>}

          <button type="submit" className="login-button" disabled={isLoading}>
            {isLoading ? 'Please wait...' : isRegisterMode ? 'Create Account' : 'Sign In'}
          </button>
        </form>

        <p className="login-toggle">
          {isRegisterMode ? 'Already have an account? ' : "Don't have an account? "}
          <button type="button" onClick={toggleMode} className="toggle-button">
            {isRegisterMode ? 'Sign In' : 'Register'}
          </button>
        </p>
      </div>
    </div>
  );
}
