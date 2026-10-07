import { store } from '../state/store.js';
import { Actions } from '../state/actions.js';
import { Toast } from '../components/toast.js';
import { patcher } from '../engine/patcher.js';

export class SalesView {
  static cart = []; // Array of { productId, name, sku, price, quantity, maxStock }

  static async render(container) {
    container.innerHTML = `
      <div class="main-content fade-in">
        <div class="view-header">
          <div>
            <h1 class="view-title">
              🛒 Point of Sale & Orders
            </h1>
            <p class="view-subtitle">Process live sales orders with atomic batch stock validation</p>
          </div>
          <div class="view-actions">
            <button id="btn-refresh-sales" class="btn btn-secondary">🔄 Refresh</button>
          </div>
        </div>

        <div class="sales-layout">
          <!-- Left: Catalog Item Quick Pick -->
          <div>
            <h3 style="margin-bottom: 12px; font-size: 1rem; color: var(--text-muted);">SELECT PRODUCTS TO ADD TO CART</h3>
            <div id="pos-grid" class="pos-catalog-grid">
              <div style="padding: 20px; color: var(--text-muted);">Loading catalog items...</div>
            </div>

            <!-- Recent Orders Section -->
            <div style="margin-top: 36px;">
              <h3 style="margin-bottom: 12px; font-size: 1.125rem;">Recent Sales Orders</h3>
              <div id="recent-orders-container">
                <div style="padding: 20px; color: var(--text-muted);">Loading orders...</div>
              </div>
            </div>
          </div>

          <!-- Right: Interactive Order Cart -->
          <div>
            <div class="card" style="position: sticky; top: 88px;">
              <h3 style="font-size: 1.125rem; margin-bottom: 16px; border-bottom: 1px solid var(--border-color); padding-bottom: 12px;">
                Current Sales Order
              </h3>

              <div class="form-group">
                <label class="form-label">Customer Name</label>
                <input type="text" id="cart-cust-name" class="form-input" placeholder="Acme Corp / John Doe" value="Client Walk-in">
              </div>

              <div class="form-group">
                <label class="form-label">Customer Email</label>
                <input type="email" id="cart-cust-email" class="form-input" placeholder="client@example.com" value="walkin@example.com">
              </div>

              <div id="cart-items-container" style="min-height: 120px; max-height: 260px; overflow-y: auto; margin-bottom: 16px;">
                <p style="color: var(--text-muted); font-size: 0.875rem; text-align: center; padding-top: 30px;">Cart is currently empty. Click a product to add.</p>
              </div>

              <div style="border-top: 1px solid var(--border-color); padding-top: 14px; margin-bottom: 16px;">
                <div style="display: flex; justify-content: space-between; font-weight: 700; font-size: 1.125rem;">
                  <span>Total Amount:</span>
                  <span id="cart-total-display">$0.00</span>
                </div>
              </div>

              <button id="btn-submit-order" class="btn btn-primary" style="width: 100%; padding: 12px;" disabled>
                ⚡ Complete Sales Order (Batch TX)
              </button>
            </div>
          </div>
        </div>
      </div>
    `;

    container.querySelector('#btn-refresh-sales').onclick = () => {
      SalesView.loadCatalog();
      SalesView.loadOrders();
    };

    container.querySelector('#btn-submit-order').onclick = () => SalesView.submitOrder();

    // Subscribe to delta changes so if an item in cart had its stock updated, we update max stock
    store.on('productDeltaApplied', ({ productId, delta }) => {
      if (delta.stock !== undefined) {
        const item = SalesView.cart.find(c => c.productId === productId);
        if (item) {
          item.maxStock = delta.stock;
          if (item.quantity > item.maxStock) {
            item.quantity = Math.max(1, item.maxStock);
            Toast.warning(`Cart quantity for ${item.name} adjusted due to real-time stock change`);
          }
          SalesView.renderCart();
        }
      }
    });

    // When order is placed, reload orders table
    store.on('orderCreated', () => {
      SalesView.loadOrders();
    });

    await SalesView.loadCatalog();
    await SalesView.loadOrders();
  }

  static async loadCatalog() {
    const grid = document.getElementById('pos-grid');
    if (!grid) return;

    try {
      const res = await Actions.fetchProducts(false); // Only active products
      const products = res.data || [];

      if (products.length === 0) {
        grid.innerHTML = `<div style="color: var(--text-muted); grid-column: 1/-1;">No products available for sale.</div>`;
        return;
      }

      grid.innerHTML = '';
      products.forEach(p => {
        const isOutOfStock = p.stock <= 0;
        const isDeleted = p.is_deleted;

        const card = document.createElement('div');
        card.className = `pos-product-card ${isOutOfStock ? 'out-of-stock' : ''} ${isDeleted ? 'card-deleted' : ''}`;
        card.dataset.productId = p.id;

        card.innerHTML = `
          <div>
            <div class="pos-card-header" style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8px;">
              <code style="font-size: 0.75rem; color: var(--text-muted);">${p.sku}</code>
              <span class="pos-stock-badge">${patcher.renderStockBadge(p.stock)}</span>
            </div>
            <h4 class="pos-card-title" style="font-size: 0.9375rem; font-weight: 600; margin-bottom: 4px;">${p.name}</h4>
            <div style="font-size: 0.75rem; color: var(--text-muted);">${p.category || 'General'}</div>
          </div>
          <div style="display: flex; justify-content: space-between; align-items: center; margin-top: 14px;">
            <div class="pos-card-price" style="font-weight: 700; font-size: 1.125rem; color: var(--primary);">$${parseFloat(p.price).toFixed(2)}</div>
            <button class="btn btn-secondary btn-sm" ${isOutOfStock || isDeleted ? 'disabled' : ''}>+ Add</button>
          </div>
        `;

        card.onclick = () => {
          if (!isOutOfStock && !isDeleted) {
            SalesView.addToCart(p);
          }
        };

        grid.appendChild(card);
      });
    } catch (err) {
      grid.innerHTML = `<div style="color: var(--danger);">Failed to load products: ${err.message}</div>`;
    }
  }

  static addToCart(product) {
    const existing = SalesView.cart.find(c => c.productId === product.id);
    if (existing) {
      if (existing.quantity < product.stock) {
        existing.quantity++;
      } else {
        Toast.warning(`Maximum available stock reached for ${product.name}`);
      }
    } else {
      if (product.stock > 0) {
        SalesView.cart.push({
          productId: product.id,
          name: product.name,
          sku: product.sku,
          price: parseFloat(product.price),
          quantity: 1,
          maxStock: product.stock,
        });
      }
    }
    SalesView.renderCart();
  }

  static renderCart() {
    const container = document.getElementById('cart-items-container');
    const totalDisplay = document.getElementById('cart-total-display');
    const submitBtn = document.getElementById('btn-submit-order');
    if (!container) return;

    if (SalesView.cart.length === 0) {
      container.innerHTML = `<p style="color: var(--text-muted); font-size: 0.875rem; text-align: center; padding-top: 30px;">Cart is currently empty. Click a product to add.</p>`;
      if (totalDisplay) totalDisplay.textContent = '$0.00';
      if (submitBtn) submitBtn.disabled = true;
      return;
    }

    let total = 0;
    let itemsHtml = '';

    SalesView.cart.forEach((item, index) => {
      const subtotal = item.quantity * item.price;
      total += subtotal;

      itemsHtml += `
        <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px 0; border-bottom: 1px solid var(--border-color);">
          <div style="flex: 1;">
            <div style="font-weight: 600; font-size: 0.875rem;">${item.name}</div>
            <div style="font-size: 0.75rem; color: var(--text-muted);">$${item.price.toFixed(2)} each</div>
          </div>
          <div style="display: flex; align-items: center; gap: 6px;">
            <button class="btn btn-secondary btn-sm" onclick="window.decCartItem(${index})">-</button>
            <span style="font-weight: 700; min-width: 20px; text-align: center;">${item.quantity}</span>
            <button class="btn btn-secondary btn-sm" onclick="window.incCartItem(${index})">+</button>
            <span style="font-weight: 600; min-width: 60px; text-align: right; margin-left: 8px;">$${subtotal.toFixed(2)}</span>
            <button class="btn btn-danger btn-sm" style="margin-left: 8px;" onclick="window.removeCartItem(${index})">&times;</button>
          </div>
        </div>
      `;
    });

    container.innerHTML = itemsHtml;
    if (totalDisplay) totalDisplay.textContent = `$${total.toFixed(2)}`;
    if (submitBtn) submitBtn.disabled = false;
  }

  static async submitOrder() {
    const custName = document.getElementById('cart-cust-name').value.trim();
    const custEmail = document.getElementById('cart-cust-email').value.trim();

    if (!custName || !custEmail) {
      alert('Please enter customer details');
      return;
    }

    if (SalesView.cart.length === 0) return;

    const payloadItems = SalesView.cart.map(c => ({
      product_id: c.productId,
      quantity: c.quantity,
    }));

    try {
      await Actions.createOrder(custName, custEmail, payloadItems);
      SalesView.cart = [];
      SalesView.renderCart();
      SalesView.loadCatalog();
      SalesView.loadOrders();
    } catch (err) {
      // Handled in Actions
    }
  }

  static async loadOrders() {
    const container = document.getElementById('recent-orders-container');
    if (!container) return;

    try {
      const res = await Actions.fetchOrders();
      const orders = res.data || [];

      if (orders.length === 0) {
        container.innerHTML = `<div class="card" style="color: var(--text-muted); padding: 20px;">No sales orders recorded yet.</div>`;
        return;
      }

      let rows = '';
      orders.forEach(o => {
        const isCancelled = o.status === 'cancelled';
        rows += `
          <tr>
            <td><code>${o.order_number}</code></td>
            <td><strong>${o.customer_name}</strong><br><small style="color:var(--text-muted);">${o.customer_email}</small></td>
            <td>${o.items ? o.items.length : 0} line items</td>
            <td style="font-weight: 700; color: var(--primary);">$${parseFloat(o.total_amount).toFixed(2)}</td>
            <td>
              <span class="badge ${isCancelled ? 'badge-danger' : 'badge-success'}">${o.status.toUpperCase()}</span>
            </td>
            <td>${new Date(o.created_at).toLocaleTimeString()}</td>
            <td style="text-align: right;">
              ${!isCancelled ? `<button class="btn btn-secondary btn-sm btn-cancel-order" data-id="${o.id}">Cancel Sale</button>` : '<span style="color:var(--text-dim); font-size:0.75rem;">Cancelled</span>'}
            </td>
          </tr>
        `;
      });

      container.innerHTML = `
        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>Order #</th>
                <th>Customer</th>
                <th>Items</th>
                <th>Total</th>
                <th>Status</th>
                <th>Time</th>
                <th style="text-align: right;">Action</th>
              </tr>
            </thead>
            <tbody>${rows}</tbody>
          </table>
        </div>
      `;

      container.querySelectorAll('.btn-cancel-order').forEach(btn => {
        btn.onclick = async () => {
          const id = btn.dataset.id;
          if (confirm('Cancel this sales order? Stock will be restored across products.')) {
            await Actions.cancelOrder(id);
            SalesView.loadOrders();
          }
        };
      });
    } catch (err) {
      container.innerHTML = `<div style="color: var(--danger);">Failed to load orders: ${err.message}</div>`;
    }
  }
}

// Global cart helper bindings
window.incCartItem = (idx) => {
  const item = SalesView.cart[idx];
  if (item && item.quantity < item.maxStock) {
    item.quantity++;
    SalesView.renderCart();
  } else {
    Toast.warning('Cannot exceed available stock');
  }
};

window.decCartItem = (idx) => {
  const item = SalesView.cart[idx];
  if (item) {
    item.quantity--;
    if (item.quantity <= 0) {
      SalesView.cart.splice(idx, 1);
    }
    SalesView.renderCart();
  }
};

window.removeCartItem = (idx) => {
  SalesView.cart.splice(idx, 1);
  SalesView.renderCart();
};
