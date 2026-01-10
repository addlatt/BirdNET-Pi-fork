import { useState, useEffect, useCallback, useRef } from 'preact/hooks';

/**
 * WebSocket hook with channel subscription support.
 * @param {string} url - WebSocket URL
 * @returns {Object} WebSocket state and methods
 */
export function useWebSocket(url) {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState(null);
  const ws = useRef(null);
  const subscribers = useRef(new Map());
  const reconnectTimeout = useRef(null);

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

        ws.current.onmessage = (event) => {
          try {
            const message = JSON.parse(event.data);
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
  const subscribe = useCallback((messageType, callback) => {
    if (!subscribers.current.has(messageType)) {
      subscribers.current.set(messageType, []);
    }
    subscribers.current.get(messageType).push(callback);

    // Return unsubscribe function
    return () => {
      const subs = subscribers.current.get(messageType);
      if (subs) {
        const index = subs.indexOf(callback);
        if (index > -1) {
          subs.splice(index, 1);
        }
      }
    };
  }, []);

  // Subscribe to a channel
  const subscribeChannel = useCallback((channel, callback) => {
    const key = `channel:${channel}`;
    if (!subscribers.current.has(key)) {
      subscribers.current.set(key, []);
    }
    subscribers.current.get(key).push(callback);

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
        const index = subs.indexOf(callback);
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
  const send = useCallback((type, payload) => {
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
