import { store } from '../state/store.js';
import { Actions } from '../state/actions.js';

export class AuditView {
  static async render(container) {
    container.innerHTML = `
      <div class="main-content fade-in">
        <div class="view-header">
          <div>
            <h1 class="view-title">
              🛡️ Audit Trail & Telemetry
            </h1>
            <p class="view-subtitle">Background worker logged mutations & real-time telemetry stream</p>
          </div>
          <div class="view-actions">
            <select id="filter-audit-entity" class="form-select" style="width: 180px;">
              <option value="">All Entities</option>
              <option value="products">Products</option>
              <option value="orders">Orders</option>
              <option value="users">Users</option>
              <option value="auth">Auth</option>
            </select>
            <button id="btn-refresh-audit" class="btn btn-secondary">🔄 Refresh</button>
          </div>
        </div>

        <!-- Telemetry Stats Cards -->
        <div class="stats-grid" id="telemetry-stats-grid">
          <div class="stat-box"><div class="stat-label">Total Requests</div><div class="stat-value" id="stat-requests">--</div></div>
          <div class="stat-box"><div class="stat-label">Active WS Clients</div><div class="stat-value" id="stat-ws" style="color:var(--success);">--</div></div>
          <div class="stat-box"><div class="stat-label">Events Broadcasted</div><div class="stat-value" id="stat-events" style="color:var(--primary);">--</div></div>
          <div class="stat-box"><div class="stat-label">Audits Logged (BG)</div><div class="stat-value" id="stat-audits" style="color:var(--warning);">--</div></div>
        </div>

        <!-- Audit Table -->
        <div id="audit-table-container">
          <div style="padding: 20px; color: var(--text-muted);">Loading audit logs...</div>
        </div>
      </div>
    `;

    container.querySelector('#btn-refresh-audit').onclick = () => {
      AuditView.loadMetrics();
      AuditView.loadAuditLogs();
    };

    const filterSelect = container.querySelector('#filter-audit-entity');
    filterSelect.onchange = () => {
      AuditView.loadAuditLogs(filterSelect.value);
    };

    // Live update when an audit event arrives via WebSocket
    store.on('auditLogged', (log) => {
      AuditView.prependLiveAuditRow(log);
      AuditView.loadMetrics();
    });

    await AuditView.loadMetrics();
    await AuditView.loadAuditLogs();
  }

  static async loadMetrics() {
    try {
      const metrics = await Actions.fetchMetrics();
      const reqEl = document.getElementById('stat-requests');
      const wsEl = document.getElementById('stat-ws');
      const evEl = document.getElementById('stat-events');
      const audEl = document.getElementById('stat-audits');

      if (reqEl) reqEl.textContent = metrics.total_requests;
      if (wsEl) wsEl.textContent = metrics.active_connections;
      if (evEl) evEl.textContent = metrics.total_events_emitted;
      if (audEl) audEl.textContent = metrics.total_audits_logged;
    } catch (err) {
      console.warn('Failed to load metrics:', err);
    }
  }

  static async loadAuditLogs(entity = '') {
    const container = document.getElementById('audit-table-container');
    if (!container) return;

    try {
      const res = await Actions.fetchAuditLogs(entity);
      const logs = res.data || [];

      if (logs.length === 0) {
        container.innerHTML = `<div class="card" style="padding: 24px; color: var(--text-muted); text-align: center;">No audit logs recorded yet.</div>`;
        return;
      }

      let rows = '';
      logs.forEach(a => {
        rows += AuditView.renderLogRow(a);
      });

      container.innerHTML = `
        <div class="table-container">
          <table class="data-table">
            <thead>
              <tr>
                <th>Timestamp</th>
                <th>Entity</th>
                <th>Action</th>
                <th>Actor Email</th>
                <th>IP / Client</th>
                <th>Changes / Payload</th>
              </tr>
            </thead>
            <tbody id="audit-table-body">${rows}</tbody>
          </table>
        </div>
      `;
    } catch (err) {
      container.innerHTML = `<div style="color: var(--danger);">Failed to load audit trail: ${err.message}</div>`;
    }
  }

  static prependLiveAuditRow(log) {
    const tbody = document.getElementById('audit-table-body');
    if (!tbody) return;

    const tr = document.createElement('tr');
    tr.className = 'pulse-update';
    tr.innerHTML = AuditView.renderLogRow(log, true);
    tbody.insertBefore(tr, tbody.firstChild);
  }

  static renderLogRow(a, isInnerOnly = false) {
    let actionBadge = `<span class="badge badge-info">${a.action}</span>`;
    if (a.action === 'CREATE') actionBadge = `<span class="badge badge-success">CREATE</span>`;
    if (a.action === 'UPDATE' || a.action === 'STOCK_ADJUST') actionBadge = `<span class="badge badge-warning">${a.action}</span>`;
    if (a.action === 'DELETE' || a.action === 'CANCEL_ORDER') actionBadge = `<span class="badge badge-danger">${a.action}</span>`;
    if (a.action === 'BATCH_TRANSACTION') actionBadge = `<span class="badge badge-info" style="background:linear-gradient(90deg,#3b82f6,#8b5cf6);">BATCH_TX</span>`;

    const summary = a.new_values || a.old_values || '-';
    const truncatedSummary = summary.length > 80 ? summary.substring(0, 80) + '...' : summary;

    const inner = `
      <td><small style="color:var(--text-muted);">${new Date(a.created_at || Date.now()).toLocaleTimeString()}</small></td>
      <td><strong>${a.entity_name}</strong></td>
      <td>${actionBadge}</td>
      <td>${a.actor_email || a.actor_id || 'System'}</td>
      <td><small style="color:var(--text-dim);">${a.ip_address || 'Internal'}</small></td>
      <td><code style="font-size:0.75rem; color:var(--text-muted);">${truncatedSummary}</code></td>
    `;

    if (isInnerOnly) return inner;
    return `<tr>${inner}</tr>`;
  }
}
