import { CONFIG } from '../config.js';

class CentralStateStore {
  constructor() {
    this.state = {
      token: localStorage.getItem(CONFIG.STORAGE_KEY_TOKEN) || null,
      user: JSON.parse(localStorage.getItem(CONFIG.STORAGE_KEY_USER) || 'null'),
      wsStatus: 'disconnected', // 'connected' | 'connecting' | 'disconnected'
      products: new Map(),       // id -> product object
      activeResourceIds: new Set(), // Set of resource IDs currently loaded/rendered in active view
      orders: [],
      auditLogs: [],
      newProductsQueue: [],      // Newly created items pending manual reload
      metrics: null,
      currentRoute: '/',
    };

    this.listeners = new Map(); // event -> Set(callback)
  }

  getState() {
    return this.state;
  }

  // Set authentication session
  setAuth(token, user) {
    this.state.token = token;
    this.state.user = user;
    if (token) {
      localStorage.setItem(CONFIG.STORAGE_KEY_TOKEN, token);
      localStorage.setItem(CONFIG.STORAGE_KEY_USER, JSON.stringify(user));
    } else {
      localStorage.removeItem(CONFIG.STORAGE_KEY_TOKEN);
      localStorage.removeItem(CONFIG.STORAGE_KEY_USER);
    }
    this.emit('authChanged', { token, user });
  }

  isAuthenticated() {
    return Boolean(this.state.token && this.state.user);
  }

  // WebSocket connection status
  setWsStatus(status) {
    if (this.state.wsStatus !== status) {
      this.state.wsStatus = status;
      this.emit('wsStatusChanged', status);
    }
  }

  // Register which items are currently being viewed
  setActiveResources(resourceType, items) {
    this.state.activeResourceIds.clear();
    if (resourceType === 'products' && Array.isArray(items)) {
      this.state.products.clear();
      items.forEach(p => {
        this.state.products.set(p.id, { ...p });
        this.state.activeResourceIds.add(p.id);
      });
    }
  }

  // Check if a given entity is currently tracked in active client status
  isResourceActive(entityId) {
    return this.state.activeResourceIds.has(entityId);
  }

  // Update a single product in state
  updateProductState(productId, delta) {
    if (this.state.products.has(productId)) {
      const current = this.state.products.get(productId);
      const updated = { ...current, ...delta };
      this.state.products.set(productId, updated);
      return updated;
    }
    return null;
  }

  // Mark product as deleted in state
  markProductDeleted(productId) {
    if (this.state.products.has(productId)) {
      const current = this.state.products.get(productId);
      current.is_deleted = true;
      this.state.products.set(productId, current);
      return current;
    }
    return null;
  }

  // Enqueue new product notification
  enqueueNewProduct(product) {
    this.state.newProductsQueue.push(product);
    this.emit('newProductAvailable', {
      count: this.state.newProductsQueue.length,
      product,
    });
  }

  clearNewProductsQueue() {
    const queue = [...this.state.newProductsQueue];
    this.state.newProductsQueue = [];
    this.emit('newProductAvailable', { count: 0 });
    return queue;
  }

  // Simple Pub/Sub for State Observers
  on(event, callback) {
    if (!this.listeners.has(event)) {
      this.listeners.set(event, new Set());
    }
    this.listeners.get(event).add(callback);
    return () => this.listeners.get(event).delete(callback);
  }

  emit(event, data) {
    if (this.listeners.has(event)) {
      this.listeners.get(event).forEach(cb => {
        try {
          cb(data);
        } catch (e) {
          console.error(`Error in state listener for ${event}:`, e);
        }
      });
    }
  }
}

export const store = new CentralStateStore();
