import { CONFIG } from '../config.js';
import { store } from './store.js';
import { Toast } from '../components/toast.js';

async function request(endpoint, options = {}) {
  const token = store.getState().token;
  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  try {
    const res = await fetch(`${CONFIG.API_BASE}${endpoint}`, {
      ...options,
      headers,
    });

    if (res.status === 401) {
      // Session expired or invalid
      store.setAuth(null, null);
      window.location.hash = '#/login';
      throw new Error('Session expired, please login again');
    }

    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      throw new Error(data.error || `HTTP error ${res.status}`);
    }

    return data;
  } catch (err) {
    Toast.error(err.message);
    throw err;
  }
}

export const Actions = {
  async register(email, fullName, password, role) {
    const res = await request('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ email, full_name: fullName, password, role }),
    });
    Toast.success('Account created successfully! Please log in.');
    return res;
  },

  async login(email, password) {
    const res = await request('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    store.setAuth(res.token, res.user);
    Toast.success(`Welcome back, ${res.user.full_name}!`);
    return res;
  },

  logout() {
    store.setAuth(null, null);
    window.location.hash = '#/login';
    Toast.info('Logged out successfully');
  },

  async fetchProducts(includeDeleted = false, limit = 100, offset = 0) {
    const res = await request(`/products?include_deleted=${includeDeleted}&limit=${limit}&offset=${offset}`);
    store.setActiveResources('products', res.data || []);
    return res;
  },

  async createProduct(productData) {
    const res = await request('/products', {
      method: 'POST',
      body: JSON.stringify(productData),
    });
    Toast.success('Product created successfully');
    return res;
  },

  async updateProduct(id, updateData) {
    const res = await request(`/products/${id}`, {
      method: 'PUT',
      body: JSON.stringify(updateData),
    });
    Toast.success('Product updated');
    return res;
  },

  async deleteProduct(id) {
    const res = await request(`/products/${id}`, {
      method: 'DELETE',
    });
    Toast.info('Product marked as deleted');
    return res;
  },

  async adjustStock(id, quantityDelta, reason = 'Manual adjustment') {
    const res = await request(`/products/${id}/stock`, {
      method: 'POST',
      body: JSON.stringify({ quantity_delta: quantityDelta, reason }),
    });
    Toast.success('Stock adjusted');
    return res;
  },

  async fetchOrders(limit = 50, offset = 0) {
    const res = await request(`/orders?limit=${limit}&offset=${offset}`);
    store.getState().orders = res.data || [];
    return res;
  },

  async createOrder(customerName, customerEmail, items) {
    const res = await request('/orders', {
      method: 'POST',
      body: JSON.stringify({
        customer_name: customerName,
        customer_email: customerEmail,
        items,
      }),
    });
    Toast.success(`Order ${res.order_number} created successfully!`);
    return res;
  },

  async cancelOrder(id) {
    const res = await request(`/orders/${id}/cancel`, {
      method: 'POST',
    });
    Toast.warning('Order cancelled and inventory restored');
    return res;
  },

  async fetchAuditLogs(entity = '', limit = 50, offset = 0) {
    const res = await request(`/audit?entity=${entity}&limit=${limit}&offset=${offset}`);
    store.getState().auditLogs = res.data || [];
    return res;
  },

  async fetchMetrics() {
    const res = await request('/metrics');
    store.getState().metrics = res;
    return res;
  },
};
