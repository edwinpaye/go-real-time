import { CONFIG } from '../config.js';
import { store } from '../state/store.js';

class WebSocketManager {
  constructor() {
    this.ws = null;
    this.reconnectAttempts = 0;
    this.reconnectTimer = null;
    this.pingTimer = null;
    this.listeners = new Set();
    this.isExplicitClose = false;

    // Listen to authentication changes
    store.on('authChanged', ({ token }) => {
      if (token) {
        this.connect();
      } else {
        this.disconnect();
      }
    });
  }

  connect() {
    const token = store.getState().token;
    if (!token) {
      store.setWsStatus('disconnected');
      return;
    }

    if (this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING)) {
      return;
    }

    this.isExplicitClose = false;
    store.setWsStatus('connecting');

    const wsUrl = `${CONFIG.WS_BASE}?token=${encodeURIComponent(token)}`;

    try {
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        this.reconnectAttempts = 0;
        store.setWsStatus('connected');
        this.startHeartbeat();
      };

      this.ws.onmessage = (event) => {
        try {
          const lines = event.data.split('\n');
          for (const line of lines) {
            if (!line.trim()) continue;
            const message = JSON.parse(line);
            this.notifyListeners(message);
          }
        } catch (err) {
          console.error('Failed to parse WebSocket frame:', err);
        }
      };

      this.ws.onerror = (err) => {
        console.warn('WebSocket error:', err);
      };

      this.ws.onclose = () => {
        this.stopHeartbeat();
        store.setWsStatus('disconnected');

        if (!this.isExplicitClose && store.getState().token) {
          this.scheduleReconnect();
        }
      };
    } catch (err) {
      console.error('Failed to establish WebSocket connection:', err);
      store.setWsStatus('disconnected');
      this.scheduleReconnect();
    }
  }

  scheduleReconnect() {
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);

    if (this.reconnectAttempts >= CONFIG.WS_MAX_RECONNECT_ATTEMPTS) {
      console.warn('Max WebSocket reconnect attempts reached');
      return;
    }

    const backoff = Math.min(1000 * Math.pow(1.5, this.reconnectAttempts), 15000);
    this.reconnectAttempts++;

    this.reconnectTimer = setTimeout(() => {
      if (store.getState().token) {
        this.connect();
      }
    }, backoff);
  }

  startHeartbeat() {
    this.stopHeartbeat();
    this.pingTimer = setInterval(() => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        this.ws.send(JSON.stringify({ action: 'ping' }));
      }
    }, 30000);
  }

  stopHeartbeat() {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
  }

  disconnect() {
    this.isExplicitClose = true;
    this.stopHeartbeat();
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
    store.setWsStatus('disconnected');
  }

  onMessage(callback) {
    this.listeners.add(callback);
    return () => this.listeners.delete(callback);
  }

  notifyListeners(message) {
    this.listeners.forEach(cb => {
      try {
        cb(message);
      } catch (e) {
        console.error('Error in WS message listener:', e);
      }
    });
  }
}

export const wsManager = new WebSocketManager();
