import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from 'react';
import type { User } from './api';

interface AuthState {
  token: string;
  user: User | null;
  isLoggedIn: boolean;
}

interface AuthContextValue extends AuthState {
  login: (token: string, user: User) => void;
  logout: () => void;
  updateBalance: (balance: number) => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>(() => {
    try {
      const token = localStorage.getItem('chain2plus1_token') || '';
      const userStr = localStorage.getItem('chain2plus1_user');
      const user = userStr ? JSON.parse(userStr) : null;
      return { token, user, isLoggedIn: !!token && !!user };
    } catch {
      return { token: '', user: null, isLoggedIn: false };
    }
  });

  const login = useCallback((token: string, user: User) => {
    localStorage.setItem('chain2plus1_token', token);
    localStorage.setItem('chain2plus1_user', JSON.stringify(user));
    setState({ token, user, isLoggedIn: true });
  }, []);

  const logout = useCallback(() => {
    localStorage.removeItem('chain2plus1_token');
    localStorage.removeItem('chain2plus1_user');
    setState({ token: '', user: null, isLoggedIn: false });
  }, []);

  const updateBalance = useCallback((balance: number) => {
    setState(prev => {
      if (!prev.user) return prev;
      const updated = { ...prev.user, balance };
      localStorage.setItem('chain2plus1_user', JSON.stringify(updated));
      return { ...prev, user: updated };
    });
  }, []);

  // Sync across tabs
  useEffect(() => {
    const handler = (e: StorageEvent) => {
      if (e.key === 'chain2plus1_token') {
        setState(prev => ({
          ...prev,
          token: e.newValue || '',
          isLoggedIn: !!e.newValue && !!localStorage.getItem('chain2plus1_user'),
        }));
      }
      if (e.key === 'chain2plus1_user') {
        try {
          const user = e.newValue ? JSON.parse(e.newValue) : null;
          setState(prev => ({ ...prev, user, isLoggedIn: !!prev.token && !!user }));
        } catch {
          // ignore
        }
      }
    };
    window.addEventListener('storage', handler);
    return () => window.removeEventListener('storage', handler);
  }, []);

  return (
    <AuthContext.Provider value={{ ...state, login, logout, updateBalance }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
