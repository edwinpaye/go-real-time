import { store } from '../state/store.js';
import { Toast } from '../components/toast.js';

class DeltaPatcher {
  // 1. Handle Real-Time Stock Changes
  handleStockChanged(eventPayload) {
    const changeEvent = eventPayload.data ? eventPayload : { entity_id: eventPayload.entity_id, delta: eventPayload.delta, data: eventPayload.data };
    const productId = changeEvent.entity_id;
    const delta = changeEvent.delta || {};
    const newStock = delta.stock !== undefined ? delta.stock : (changeEvent.data && changeEvent.data.stock);

    // CRITICAL: Check if this item is currently in client status (requested/loaded previously)
    if (!store.isResourceActive(productId)) {
      // Not loaded or requested previously in active view: DO NOT touch DOM or re-request data
      return;
    }

    // 1. Update State Store
    store.updateProductState(productId, { stock: newStock });

    // 2. Fine-grained DOM patch on Active Table Rows
    const row = document.querySelector(`tr[data-product-id="${productId}"]`);
    if (row) {
      const stockCell = row.querySelector('.cell-stock');
      if (stockCell) {
        stockCell.innerHTML = this.renderStockBadge(newStock);
      }
      // Apply subtle visual pulse animation
      row.classList.remove('pulse-stock-change');
      void row.offsetWidth; // Force reflow
      row.classList.add('pulse-stock-change');
    }

    // 3. Fine-grained DOM patch on POS Product Cards if present
    const card = document.querySelector(`.pos-product-card[data-product-id="${productId}"]`);
    if (card) {
      const cardStock = card.querySelector('.pos-stock-badge');
      if (cardStock) {
        cardStock.innerHTML = this.renderStockBadge(newStock);
      }
      if (newStock <= 0) {
        card.classList.add('out-of-stock');
      } else {
        card.classList.remove('out-of-stock');
      }
      card.classList.remove('pulse-stock-change');
      void card.offsetWidth;
      card.classList.add('pulse-stock-change');
    }

    // Emit event for custom subscribers
    store.emit('productDeltaApplied', { productId, delta: { stock: newStock } });
  }

  // 2. Handle Item Updates (Name, Price, Category, etc.)
  handleProductUpdated(eventPayload) {
    const changeEvent = eventPayload.data ? eventPayload : { entity_id: eventPayload.entity_id, delta: eventPayload.delta, data: eventPayload.data };
    const productId = changeEvent.entity_id;
    const data = changeEvent.data || {};
    const delta = changeEvent.delta || {};

    if (!store.isResourceActive(productId)) {
      return; // Skip DOM patch if not currently active
    }

    // Update state
    const updated = store.updateProductState(productId, delta);

    // Patch Table Row
    const row = document.querySelector(`tr[data-product-id="${productId}"]`);
    if (row) {
      if (delta.name !== undefined) {
        const nameCell = row.querySelector('.cell-name');
        if (nameCell) nameCell.textContent = delta.name;
      }
      if (delta.category !== undefined) {
        const catCell = row.querySelector('.cell-category');
        if (catCell) catCell.textContent = delta.category;
      }
      if (delta.price !== undefined) {
        const priceCell = row.querySelector('.cell-price');
        if (priceCell) priceCell.textContent = `$${parseFloat(delta.price).toFixed(2)}`;
      }
      if (delta.min_stock !== undefined) {
        const minStockCell = row.querySelector('.cell-min-stock');
        if (minStockCell) minStockCell.textContent = delta.min_stock;
      }

      row.classList.remove('pulse-update');
      void row.offsetWidth;
      row.classList.add('pulse-update');
    }

    // Patch POS Card if present
    const card = document.querySelector(`.pos-product-card[data-product-id="${productId}"]`);
    if (card) {
      if (delta.name !== undefined) {
        const title = card.querySelector('.pos-card-title');
        if (title) title.textContent = delta.name;
      }
      if (delta.price !== undefined) {
        const price = card.querySelector('.pos-card-price');
        if (price) price.textContent = `$${parseFloat(delta.price).toFixed(2)}`;
      }
    }

    store.emit('productDeltaApplied', { productId, delta, data: updated });
  }

  // 3. Handle Item Deleted: DANGER STYLES & UNCLICKABLE IN GUI
  handleProductDeleted(eventPayload) {
    const productId = eventPayload.entity_id;

    if (!store.isResourceActive(productId)) {
      return;
    }

    // Mark as deleted in state store
    store.markProductDeleted(productId);

    // Patch Table Row to Danger & Unclickable State
    const row = document.querySelector(`tr[data-product-id="${productId}"]`);
    if (row) {
      row.classList.add('row-deleted-danger');

      // Update status badge to DELETED
      const statusCell = row.querySelector('.cell-status');
      if (statusCell) {
        statusCell.innerHTML = `<span class="badge badge-danger">DELETED</span>`;
      }

      // Disable all action buttons and links
      const buttons = row.querySelectorAll('button, a, input');
      buttons.forEach(btn => {
        btn.setAttribute('disabled', 'true');
        btn.style.pointerEvents = 'none';
        btn.style.cursor = 'not-allowed';
      });
    }

    // Patch POS Card to Danger & Unclickable State
    const card = document.querySelector(`.pos-product-card[data-product-id="${productId}"]`);
    if (card) {
      card.classList.add('card-deleted');
      const badgeContainer = card.querySelector('.pos-card-header');
      if (badgeContainer) {
        badgeContainer.innerHTML += `<span class="badge badge-danger">UNAVAILABLE</span>`;
      }
    }

    store.emit('productDeleted', { productId });
  }

  // 4. Handle Item Created: SHOW NOTIFICATION TO REFRESH MANUALLY
  handleProductCreated(eventPayload) {
    const newProduct = eventPayload.data || eventPayload;

    // Enqueue new product in state
    store.enqueueNewProduct(newProduct);

    // If currently on products catalog view, show or update the floating alert banner
    this.renderNewItemBanner(newProduct);

    // Toast alert
    Toast.info(`New item added: "${newProduct.name || 'Product'}". Click refresh to view.`);
  }

  renderNewItemBanner(product) {
    const container = document.getElementById('new-item-alert-container');
    if (!container) return;

    const count = store.getState().newProductsQueue.length;
    container.innerHTML = `
      <div class="new-item-alert-banner">
        <div>
          <strong>🚀 ${count} new item${count > 1 ? 's' : ''} added</strong> in catalog by another session.
        </div>
        <button id="btn-manual-refresh-catalog">Refresh Catalog</button>
      </div>
    `;

    const refreshBtn = document.getElementById('btn-manual-refresh-catalog');
    if (refreshBtn) {
      refreshBtn.onclick = () => {
        store.clearNewProductsQueue();
        container.innerHTML = '';
        store.emit('requestRefreshCatalog', {});
      };
    }
  }

  // 5. Handle Organized Batch Transactions (e.g. Sales Order processing)
  handleBatchTransaction(batch) {
    if (!batch || !Array.isArray(batch.changes)) return;

    // Apply all atomic changes in a single loop
    batch.changes.forEach(change => {
      switch (change.type) {
        case 'STOCK_CHANGED':
          this.handleStockChanged(change);
          break;
        case 'ENTITY_UPDATED':
          if (change.resource === 'products') {
            this.handleProductUpdated(change);
          }
          break;
        case 'ENTITY_DELETED':
          if (change.resource === 'products') {
            this.handleProductDeleted(change);
          }
          break;
        case 'ENTITY_CREATED':
          if (change.resource === 'orders') {
            this.handleOrderCreated(change);
          } else if (change.resource === 'products') {
            this.handleProductCreated(change);
          }
          break;
      }
    });

    Toast.info(`⚡ Batch update processed: ${batch.operation || 'Transaction'} (${batch.changes.length} changes)`);
  }

  handleOrderCreated(eventPayload) {
    const order = eventPayload.data || eventPayload;
    store.emit('orderCreated', order);
  }

  handleAuditLogCreated(eventPayload) {
    const log = eventPayload.data || eventPayload;
    store.emit('auditLogged', log);
  }

  renderStockBadge(stock) {
    if (stock <= 0) {
      return `<span class="badge badge-danger">Out of Stock (0)</span>`;
    }
    if (stock < 10) {
      return `<span class="badge badge-warning">Low: ${stock} left</span>`;
    }
    return `<span class="badge badge-success">${stock} in stock</span>`;
  }
}

export const patcher = new DeltaPatcher();
