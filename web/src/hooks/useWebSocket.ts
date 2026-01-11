import { useState, useEffect, useCallback, useRef } from 'preact/hooks';
import type { WSMessage, WSMessageType } from '../types/api';

/**
 * Callback type for message handlers
 */
type MessageCallback<T = unknown> = (payload: T, message: WSMessage<T>) => void;

/**
 * WebSocket hook return type
 */
interface UseWebSocketReturn {
  isConnected: boolean;
  lastMessage: WSMessage | null;
  subscribe: <T = unknown>(messageType: WSMessageType | string, callback: MessageCallback<T>) => () => void;
  subscribeChannel: <T = unknown>(channel: string, callback: MessageCallback<T>) => () => void;
  send: <T = unknown>(type: string, payload: T) => void;
}

/**
 * WebSocket hook with channel subscription support.
 * @param url - WebSocket URL
 * @returns WebSocket state and methods
 */
export function useWebSocket(url: string): UseWebSocketReturn {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null);
  const ws = useRef<WebSocket | null>(null);
  const subscribers = useRef<Map<string, MessageCallback[]>>(new Map());
  const reconnectTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Connect to WebSocket
  useEffect(() => {
    const connect = () => {
      try {
        ws.current = new WebSocket(url);

        ws.current.onopen = () => {
          console.log('WebSocket connected');
          setIsConnected(true);
        };

        ws.current.onclose = () => {
          console.log('WebSocket disconnected');
          setIsConnected(false);

          // Attempt to reconnect after 3 seconds
          reconnectTimeout.current = setTimeout(() => {
            console.log('Attempting to reconnect...');
            connect();
          }, 3000);
        };

        ws.current.onerror = (error) => {
          console.error('WebSocket error:', error);
        };

        ws.current.onmessage = (event: MessageEvent) => {
          try {
            const message = JSON.parse(event.data) as WSMessage;
            setLastMessage(message);

            // Notify type subscribers
            const typeSubs = subscribers.current.get(message.type) || [];
            typeSubs.forEach((callback) => callback(message.payload, message));

            // Notify channel subscribers if channel is specified
            if (message.channel) {
              const channelKey = `channel:${message.channel}`;
              const channelSubs = subscribers.current.get(channelKey) || [];
              channelSubs.forEach((callback) => callback(message.payload, message));
            }
          } catch (e) {
            console.error('Failed to parse WebSocket message:', e);
          }
        };
      } catch (error) {
        console.error('Failed to create WebSocket:', error);
      }
    };

    connect();

    return () => {
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }
      if (ws.current) {
        ws.current.close();
      }
    };
  }, [url]);

  // Subscribe to a message type
  const subscribe = useCallback(<T = unknown>(messageType: WSMessageType | string, callback: MessageCallback<T>) => {
    if (!subscribers.current.has(messageType)) {
      subscribers.current.set(messageType, []);
    }
    subscribers.current.get(messageType)!.push(callback as MessageCallback);

    // Return unsubscribe function
    return () => {
      const subs = subscribers.current.get(messageType);
      if (subs) {
        const index = subs.indexOf(callback as MessageCallback);
        if (index > -1) {
          subs.splice(index, 1);
        }
      }
    };
  }, []);

  // Subscribe to a channel
  const subscribeChannel = useCallback(<T = unknown>(channel: string, callback: MessageCallback<T>) => {
    const key = `channel:${channel}`;
    if (!subscribers.current.has(key)) {
      subscribers.current.set(key, []);
    }
    subscribers.current.get(key)!.push(callback as MessageCallback);

    // Send subscription message to server
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify({
        type: 'subscribe',
        payload: { channels: [channel] },
      }));
    }

    // Return unsubscribe function
    return () => {
      const subs = subscribers.current.get(key);
      if (subs) {
        const index = subs.indexOf(callback as MessageCallback);
        if (index > -1) {
          subs.splice(index, 1);
        }
      }

      // Send unsubscription message to server
      if (ws.current && ws.current.readyState === WebSocket.OPEN) {
        ws.current.send(JSON.stringify({
          type: 'unsubscribe',
          payload: { channels: [channel] },
        }));
      }
    };
  }, []);

  // Send a message
  const send = useCallback(<T = unknown>(type: string, payload: T) => {
    if (ws.current && ws.current.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify({ type, payload }));
    }
  }, []);

  return {
    isConnected,
    lastMessage,
    subscribe,
    subscribeChannel,
    send,
  };
}
