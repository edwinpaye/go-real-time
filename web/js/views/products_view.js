import { store } from '../state/store.js';
import { Actions } from '../state/actions.js';
import { Modal } from '../components/modal.js';
import { patcher } from '../engine/patcher.js';

export class ProductsView {
  static async render(container) {
    container.innerHTML = `
      <div class="main-content fade-in">
        <div class="view-header">
          <div>
            <h1 class="view-title">
              📦 Product Inventory & Stock Catalog
            </h1>
            <p class="view-subtitle">Real-time WebSocket synchronized inventory</p>
          </div>
          <div class="view-actions">
            <button id="btn-create-product" class="btn btn-primary">+ Add New Product</button>
            <button id="btn-refresh-products" class="btn btn-secondary">🔄 Refresh</button>
          </div>
        </div>

        <!-- Container for New Items WebSocket Announcement Banner -->
        <div id="new-item-alert-container"></div>

        <div id="products-table-container">
          <div style="padding: 40px; text-align: center; color: var(--text-muted);">Loading inventory items...</div>
        </div>
      </div>
    `;

    // Bind header buttons
    container.querySelector('#btn-create-product').onclick = () => ProductsView.openCreateModal();
    container.querySelector('#btn-refresh-products').onclick = () => ProductsView.loadData();

    // Listen for manual refresh requested from banner
    store.on('requestRefreshCatalog', () => {
      ProductsView.loadData();
    });

    await ProductsView.loadData();
  }

  static async loadData() {
    const tableContainer = document.getElementById('products-table-container');
    if (!tableContainer) return;

    try {
      const res = await Actions.fetchProducts(true); // Include soft-deleted to demonstrate unclickable danger state
      const products = res.data || [];

      if (products.length === 0) {
        tableContainer.innerHTML = `
          <div class="card" style="text-align: center; padding: 40px; color: var(--text-muted);">
            No products found in catalog. Click "+ Add New Product" to create your first item.
          </div>
        `;
        return;
      }

      let rowsHtml = '';
      products.forEach(p => {
        const isDeleted = p.is_deleted;
        const rowClass = isDeleted ? 'row-deleted-danger' : '';

        rowsHtml += `
          <tr data-product-id="${p.id}" class="${rowClass}">
            <td class="cell-sku"><code>${p.sku}</code></td>
            <td class="cell-name item-name"><strong>${p.name}</strong></td>
            <td class="cell-category"><span class="badge badge-muted">${p.category || 'General'}</span></td>
            <td class="cell-price">$${parseFloat(p.price).toFixed(2)}</td>
            <td class="cell-stock">${patcher.renderStockBadge(p.stock)}</td>
            <td class="cell-min-stock">${p.min_stock}</td>
            <td class="cell-status">
              ${isDeleted ? '<span class="badge badge-danger">DELETED</span>' : '<span class="badge badge-success">ACTIVE</span>'}
            </td>
            <td style="text-align: right;">
              <button class="btn btn-secondary btn-sm btn-adjust-stock" data-id="${p.id}" ${isDeleted ? 'disabled' : ''}>Adjust Stock</button>
              <button class="btn btn-secondary btn-sm btn-edit-product" data-id="${p.id}" ${isDeleted ? 'disabled' : ''}>Edit</button>
              <button class="btn btn-danger btn-sm btn-delete-product" data-id="${p.id}" ${isDeleted ? 'disabled' : ''}>Delete</button>
            </td>
          </tr>
        `;
      });

      tableContainer.innerHTML = `
        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>SKU</th>
                <th>Product Name</th>
                <th>Category</th>
                <th>Price</th>
                <th>Current Stock</th>
                <th>Min Alert</th>
                <th>Status</th>
                <th style="text-align: right;">Actions</th>
              </tr>
            </thead>
            <tbody id="products-table-body">
              ${rowsHtml}
            </tbody>
          </table>
        </div>
      `;

      ProductsView.bindTableEvents(tableContainer);
    } catch (err) {
      tableContainer.innerHTML = `
        <div class="card" style="color: var(--danger); padding: 20px;">
          Failed to load products: ${err.message}
        </div>
      `;
    }
  }

  static bindTableEvents(container) {
    container.querySelectorAll('.btn-adjust-stock').forEach(btn => {
      btn.onclick = () => {
        const id = btn.dataset.id;
        ProductsView.openAdjustStockModal(id);
      };
    });

    container.querySelectorAll('.btn-edit-product').forEach(btn => {
      btn.onclick = () => {
        const id = btn.dataset.id;
        ProductsView.openEditModal(id);
      };
    });

    container.querySelectorAll('.btn-delete-product').forEach(btn => {
      btn.onclick = async () => {
        const id = btn.dataset.id;
        if (confirm('Are you sure you want to delete this product? Real-time change will be broadcasted.')) {
          await Actions.deleteProduct(id);
        }
      };
    });
  }

  static openCreateModal() {
    Modal.open({
      title: 'Add New Product',
      contentHtml: `
        <form id="form-create-product">
          <div class="form-group">
            <label class="form-label">SKU</label>
            <input type="text" id="p-sku" class="form-input" placeholder="PROD-001" required>
          </div>
          <div class="form-group">
            <label class="form-label">Product Name</label>
            <input type="text" id="p-name" class="form-input" placeholder="Ergonomic Wireless Mouse" required>
          </div>
          <div class="form-group">
            <label class="form-label">Category</label>
            <input type="text" id="p-category" class="form-input" placeholder="Hardware" value="Electronics">
          </div>
          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
            <div class="form-group">
              <label class="form-label">Price ($)</label>
              <input type="number" step="0.01" id="p-price" class="form-input" placeholder="49.99" required min="0">
            </div>
            <div class="form-group">
              <label class="form-label">Initial Stock</label>
              <input type="number" id="p-stock" class="form-input" placeholder="50" required min="0">
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">Min Stock Threshold</label>
            <input type="number" id="p-min-stock" class="form-input" value="5" min="0">
          </div>
          <div class="form-group">
            <label class="form-label">Description</label>
            <textarea id="p-desc" class="form-textarea" rows="2" placeholder="Item details..."></textarea>
          </div>
        </form>
      `,
      buttons: [
        { label: 'Cancel', className: 'btn-secondary', onClick: (e, modal) => modal.close() },
        {
          label: 'Save Product',
          className: 'btn-primary',
          onClick: async (e, modal) => {
            const sku = document.getElementById('p-sku').value.trim();
            const name = document.getElementById('p-name').value.trim();
            const category = document.getElementById('p-category').value.trim();
            const price = parseFloat(document.getElementById('p-price').value);
            const stock = parseInt(document.getElementById('p-stock').value, 10);
            const minStock = parseInt(document.getElementById('p-min-stock').value, 10);
            const desc = document.getElementById('p-desc').value.trim();

            if (!sku || !name || isNaN(price) || isNaN(stock)) {
              alert('Please complete all required fields properly');
              return;
            }

            try {
              await Actions.createProduct({
                sku,
                name,
                category,
                price,
                stock,
                min_stock: minStock,
                description: desc,
              });
              modal.close();
              ProductsView.loadData();
            } catch (err) {
              // Handled in Actions
            }
          },
        },
      ],
    });
  }

  static openAdjustStockModal(id) {
    const product = store.getState().products.get(id);
    if (!product) return;

    Modal.open({
      title: `Adjust Stock: ${product.name}`,
      contentHtml: `
        <div style="margin-bottom: 16px;">
          <p>Current Stock: <strong>${product.stock}</strong></p>
        </div>
        <div class="form-group">
          <label class="form-label">Stock Change Delta (e.g. +10 or -5)</label>
          <input type="number" id="adj-delta" class="form-input" placeholder="+5" required>
        </div>
        <div class="form-group">
          <label class="form-label">Reason</label>
          <input type="text" id="adj-reason" class="form-input" placeholder="Restock shipment #8841" value="Inventory adjustment">
        </div>
      `,
      buttons: [
        { label: 'Cancel', className: 'btn-secondary', onClick: (e, modal) => modal.close() },
        {
          label: 'Apply Delta',
          className: 'btn-primary',
          onClick: async (e, modal) => {
            const delta = parseInt(document.getElementById('adj-delta').value, 10);
            const reason = document.getElementById('adj-reason').value.trim();

            if (isNaN(delta) || delta === 0) {
              alert('Please specify a non-zero integer delta');
              return;
            }

            try {
              await Actions.adjustStock(id, delta, reason);
              modal.close();
            } catch (err) {
              // Handled in Actions
            }
          },
        },
      ],
    });
  }

  static openEditModal(id) {
    const product = store.getState().products.get(id);
    if (!product) return;

    Modal.open({
      title: `Edit Product: ${product.name}`,
      contentHtml: `
        <form id="form-edit-product">
          <div class="form-group">
            <label class="form-label">Product Name</label>
            <input type="text" id="edit-name" class="form-input" value="${product.name}" required>
          </div>
          <div class="form-group">
            <label class="form-label">Category</label>
            <input type="text" id="edit-category" class="form-input" value="${product.category || ''}">
          </div>
          <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
            <div class="form-group">
              <label class="form-label">Price ($)</label>
              <input type="number" step="0.01" id="edit-price" class="form-input" value="${product.price}" required min="0">
            </div>
            <div class="form-group">
              <label class="form-label">Min Stock Threshold</label>
              <input type="number" id="edit-min-stock" class="form-input" value="${product.min_stock}" min="0">
            </div>
          </div>
          <div class="form-group">
            <label class="form-label">Description</label>
            <textarea id="edit-desc" class="form-textarea" rows="2">${product.description || ''}</textarea>
          </div>
        </form>
      `,
      buttons: [
        { label: 'Cancel', className: 'btn-secondary', onClick: (e, modal) => modal.close() },
        {
          label: 'Save Changes',
          className: 'btn-primary',
          onClick: async (e, modal) => {
            const name = document.getElementById('edit-name').value.trim();
            const category = document.getElementById('edit-category').value.trim();
            const price = parseFloat(document.getElementById('edit-price').value);
            const minStock = parseInt(document.getElementById('edit-min-stock').value, 10);
            const desc = document.getElementById('edit-desc').value.trim();

            if (!name || isNaN(price)) {
              alert('Please fill out required fields properly');
              return;
            }

            try {
              await Actions.updateProduct(id, {
                name,
                category,
                price,
                min_stock: minStock,
                description: desc,
              });
              modal.close();
            } catch (err) {
              // Handled in Actions
            }
          },
        },
      ],
    });
  }
}
