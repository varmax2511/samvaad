import { createContext, useContext, useState } from 'react';
import type { ReactNode } from 'react';
import type { User } from '../types';
import * as authService from '../services/auth';

interface AuthContextType {
  user: User | null;
  login: (username: string, password: string) => Promise<void>;
  register: (username: string, password: string) => Promise<void>;
  logout: () => void;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => {
    const stored = sessionStorage.getItem('samvaad_user');
    return stored ? JSON.parse(stored) : null;
  });

  const login = async (username: string, password: string) => {
    const response = await authService.login({ username, password });

    if (!response.token || !response.userId) {
      throw new Error('Invalid response from server');
    }

    const newUser: User = {
      id: response.userId,
      username,
      token: response.token,
    };
    setUser(newUser);
    sessionStorage.setItem('samvaad_user', JSON.stringify(newUser));
  };

  const register = async (username: string, password: string) => {
    const response = await authService.register({ username, password });

    if (!response.token || !response.userId) {
      throw new Error('Invalid response from server');
    }

    const newUser: User = {
      id: response.userId,
      username,
      token: response.token,
    };
    setUser(newUser);
    sessionStorage.setItem('samvaad_user', JSON.stringify(newUser));
  };

  const logout = () => {
    setUser(null);
    sessionStorage.removeItem('samvaad_user');
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        login,
        register,
        logout,
        isAuthenticated: user !== null,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
