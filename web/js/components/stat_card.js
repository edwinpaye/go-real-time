// Metric Stat Card Component

export class StatCard {
  static create(label, value, extraText = '') {
    const box = document.createElement('div');
    box.className = 'stat-box';
    box.innerHTML = `
      <div class="stat-label">${label}</div>
      <div class="stat-value">${value}</div>
      ${extraText ? `<div style="font-size:0.75rem; color:var(--text-dim); margin-top:4px;">${extraText}</div>` : ''}
    `;
    return box;
  }
}
