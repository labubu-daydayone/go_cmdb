import { useEffect, useRef, useCallback, useState } from 'react';
import { WebSocketMessage, UseWebSocketOptions } from './useWebSocket';

/**
 * Mock WebSocket Hook
 * 用于前端开发时模拟WebSocket连接
 */
export const useMockWebSocket = (token: string | null, options: UseWebSocketOptions = {}) => {
  const {
    onMessage,
    onConnect,
    onDisconnect,
    onError,
  } = options;

  const [isConnected, setIsConnected] = useState(false);
  const [subscriptions, setSubscriptions] = useState<Set<string>>(new Set());
  const intervalRef = useRef<NodeJS.Timeout | null>(null);

  const connect = useCallback(() => {
    if (!token) return;

    console.log('[Mock WebSocket] Connecting...');
    
    // 模拟连接延迟
    setTimeout(() => {
      console.log('[Mock WebSocket] Connected');
      setIsConnected(true);
      onConnect?.();
    }, 100);
  }, [token, onConnect]);

  const subscribe = useCallback((channel: string) => {
    console.log('[Mock WebSocket] Subscribing to channel:', channel);
    setSubscriptions((prev) => new Set(prev).add(channel));
  }, []);

  const unsubscribe = useCallback((channel: string) => {
    console.log('[Mock WebSocket] Unsubscribing from channel:', channel);
    setSubscriptions((prev) => {
      const newSet = new Set(prev);
      newSet.delete(channel);
      return newSet;
    });
  }, []);

  const disconnect = useCallback(() => {
    console.log('[Mock WebSocket] Disconnecting...');
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
    }
    setIsConnected(false);
    onDisconnect?.();
  }, [onDisconnect]);

  // 模拟定期发送消息
  useEffect(() => {
    if (!isConnected || subscriptions.size === 0) return;

    // 每5秒发送一次模拟消息
    intervalRef.current = setInterval(() => {
      subscriptions.forEach((channel) => {
        const mockMessage: WebSocketMessage = {
          type: 'list_update',
          action: 'update',
          resource: channel.split(':')[0] || 'unknown',
          resource_id: String(Math.floor(Math.random() * 100)),
          data: {
            message: `Mock update for ${channel}`,
            timestamp: new Date().toISOString(),
          },
          timestamp: Date.now(),
          user_id: '1',
        };

        console.log('[Mock WebSocket] Sending message:', mockMessage);
        onMessage?.(mockMessage);
      });
    }, 5000);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
      }
    };
  }, [isConnected, subscriptions, onMessage]);

  useEffect(() => {
    if (token) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [token, connect, disconnect]);

  return {
    isConnected,
    subscribe,
    unsubscribe,
    disconnect,
    reconnect: connect,
  };
};
