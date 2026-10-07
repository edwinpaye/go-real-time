// Reusable Modal Component

class ModalManager {
  constructor() {
    this.overlay = null;
    this.titleEl = null;
    this.bodyEl = null;
    this.footerEl = null;
    this.init();
  }

  init() {
    this.overlay = document.createElement('div');
    this.overlay.className = 'modal-overlay';
    this.overlay.innerHTML = `
      <div class="modal-content">
        <div class="modal-header">
          <h3 id="modal-title">Dialog</h3>
          <button class="modal-close-btn" id="modal-close-x">&times;</button>
        </div>
        <div class="modal-body" id="modal-body"></div>
        <div class="modal-footer" id="modal-footer"></div>
      </div>
    `;
    document.body.appendChild(this.overlay);

    this.titleEl = this.overlay.querySelector('#modal-title');
    this.bodyEl = this.overlay.querySelector('#modal-body');
    this.footerEl = this.overlay.querySelector('#modal-footer');

    this.overlay.querySelector('#modal-close-x').onclick = () => this.close();
    this.overlay.onclick = (e) => {
      if (e.target === this.overlay) this.close();
    };
  }

  open({ title, contentHtml, buttons = [] }) {
    this.titleEl.textContent = title || '';
    this.bodyEl.innerHTML = contentHtml || '';
    this.footerEl.innerHTML = '';

    buttons.forEach(btn => {
      const button = document.createElement('button');
      button.className = `btn ${btn.className || 'btn-secondary'}`;
      button.textContent = btn.label;
      button.onclick = (e) => {
        if (btn.onClick) {
          btn.onClick(e, this);
        } else {
          this.close();
        }
      };
      this.footerEl.appendChild(button);
    });

    this.overlay.classList.add('active');
  }

  close() {
    this.overlay.classList.remove('active');
  }

  getBody() {
    return this.bodyEl;
  }
}

export const Modal = new ModalManager();
