import { useEffect, useRef, useCallback, useState } from 'react';

interface WSService {
  isConnected: boolean;
  connect: () => void;
  disconnect: () => void;
}

export function useWebSocket(): WSService {
  const [isConnected, setIsConnected] = useState<boolean>(false);
  const wsRef = useRef<WebSocket | null>(null);
  const retryTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const retryCount = useRef<number>(0);

  const connect = useCallback(() => {
    const token = localStorage.getItem('chain2plus1_token');
    if (!token) return;

    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    const url = `${proto}//${host}/ws?token=${token}`;

    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      setIsConnected(true);
      retryCount.current = 0;
    };

    ws.onclose = () => {
      setIsConnected(false);
      // Exponential backoff reconnect
      const delay = Math.min(1000 * Math.pow(2, retryCount.current), 30000);
      retryCount.current++;
      retryTimer.current = setTimeout(() => connect(), delay);
    };

    ws.onerror = () => {
      ws.close();
    };
  }, []);

  const disconnect = useCallback(() => {
    clearTimeout(retryTimer.current);
    wsRef.current?.close();
    wsRef.current = null;
    setIsConnected(false);
  }, []);

  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  const svc: WSService = { isConnected, connect, disconnect };
  return svc;
}
