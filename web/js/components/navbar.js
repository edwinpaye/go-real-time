import { store } from '../state/store.js';
import { Actions } from '../state/actions.js';

export class Navbar {
  static render(container) {
    const isAuth = store.isAuthenticated();
    const user = store.getState().user;
    const wsStatus = store.getState().wsStatus;
    const currentHash = window.location.hash || '#/products';

    if (!isAuth) {
      container.innerHTML = `
        <nav class="navbar">
          <div class="nav-brand">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
            </svg>
            <span>OmniSales Pro</span>
          </div>
          <div class="nav-links">
            <a href="#/login" class="nav-item ${currentHash === '#/login' ? 'active' : ''}">Login</a>
            <a href="#/register" class="nav-item ${currentHash === '#/register' ? 'active' : ''}">Register</a>
          </div>
        </nav>
      `;
      return;
    }

    container.innerHTML = `
      <nav class="navbar">
        <div class="nav-brand">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
          </svg>
          <span>OmniSales Pro</span>
        </div>

        <div class="nav-links">
          <a href="#/products" class="nav-item ${currentHash === '#/products' ? 'active' : ''}">Products & Stock</a>
          <a href="#/sales" class="nav-item ${currentHash === '#/sales' ? 'active' : ''}">Sales POS & Orders</a>
          <a href="#/audit" class="nav-item ${currentHash === '#/audit' ? 'active' : ''}">Audit Trail</a>
        </div>

        <div class="nav-right">
          <!-- Real-time WebSocket connection state indicator -->
          <div id="ws-indicator" class="ws-status-badge ${wsStatus}">
            <span class="status-dot"></span>
            <span id="ws-text">${wsStatus.toUpperCase()}</span>
          </div>

          <div style="font-size: 0.875rem; color: var(--text-muted);">
            <strong style="color: #fff;">${user ? user.full_name : ''}</strong> 
            <span class="badge badge-info" style="margin-left: 4px;">${user ? user.role : ''}</span>
          </div>

          <button id="nav-logout-btn" class="btn btn-secondary btn-sm">Logout</button>
        </div>
      </nav>
    `;

    const logoutBtn = container.querySelector('#nav-logout-btn');
    if (logoutBtn) {
      logoutBtn.onclick = () => Actions.logout();
    }
  }

  static updateWsStatus(status) {
    const indicator = document.getElementById('ws-indicator');
    const text = document.getElementById('ws-text');
    if (indicator && text) {
      indicator.className = `ws-status-badge ${status}`;
      text.textContent = status.toUpperCase();
    }
  }
}
