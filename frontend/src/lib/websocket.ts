import { useEffect, useRef, useCallback, useState } from 'react';
import { playNotificationSound } from './notification';

interface WSEvent {
  type: string;
  title: string;
  message: string;
  data?: any;
}

interface WSService {
  isConnected: boolean;
  connect: () => void;
  disconnect: () => void;
  onEvent: (handler: (event: WSEvent) => void) => () => void;
}

type EventHandler = (event: WSEvent) => void;
const handlers = new Set<EventHandler>();

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

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        const wsEvent: WSEvent = {
          type: data.type || 'notification',
          title: data.title || '',
          message: data.message || '',
          data: data.data,
        };
        // Play sound and notify handlers
        if (data.title || data.message) {
          playNotificationSound('info');
        }
        handlers.forEach(h => h(wsEvent));
      } catch {
        // Not JSON, ignore
      }
    };

    ws.onclose = () => {
      setIsConnected(false);
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

  const onEvent = useCallback((handler: EventHandler): (() => void) => {
    handlers.add(handler);
    return () => { handlers.delete(handler); };
  }, []);

  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

  return { isConnected, connect, disconnect, onEvent };
}
