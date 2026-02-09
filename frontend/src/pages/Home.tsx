import { useState } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import './Home.css';

export default function Home() {
  const [roomId, setRoomId] = useState('');
  const [error, setError] = useState('');
  const { user, logout } = useAuth();
  const navigate = useNavigate();

  const generateRoomId = () => {
    return Math.random().toString(36).substring(2, 8).toUpperCase();
  };

  const handleJoinRoom = (e: FormEvent) => {
    e.preventDefault();
    setError('');

    if (!roomId.trim()) {
      setError('Please enter a room code');
      return;
    }

    navigate(`/call/${roomId.trim()}`);
  };

  const handleStartCall = () => {
    const newRoomId = generateRoomId();
    navigate(`/call/${newRoomId}`);
  };

  const handleLogout = () => {
    logout();
    navigate('/');
  };

  return (
    <div className="home-container">
      <header className="home-header">
        <span className="user-name">{user?.username}</span>
        <button onClick={handleLogout} className="logout-button">
          Log out
        </button>
      </header>

      <main className="home-main">
        <div className="home-card">
          <h1 className="home-title">Start a conversation</h1>

          <div className="home-actions">
            <button onClick={handleStartCall} className="start-button">
              New Call
            </button>

            <div className="divider">
              <span>or join existing</span>
            </div>

            <form onSubmit={handleJoinRoom} className="join-form">
              <input
                type="text"
                value={roomId}
                onChange={(e) => setRoomId(e.target.value.toUpperCase())}
                placeholder="Enter room code"
                className="room-input"
                maxLength={10}
              />
              <button type="submit" className="join-button">
                Join
              </button>
            </form>

            {error && <p className="home-error">{error}</p>}
          </div>
        </div>
      </main>
    </div>
  );
}
