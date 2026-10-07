import { Actions } from '../state/actions.js';

export class RegisterView {
  static render(container) {
    container.innerHTML = `
      <div class="auth-wrapper fade-in">
        <div class="auth-card">
          <div class="auth-header">
            <h2>Create Account</h2>
            <p style="color: var(--text-muted); font-size: 0.875rem; margin-top: 4px;">Register a new user in the enterprise sales platform</p>
          </div>

          <form id="register-form">
            <div class="form-group">
              <label class="form-label">Full Name</label>
              <input type="text" id="reg-name" class="form-input" placeholder="Sarah Jenkins" required>
            </div>

            <div class="form-group">
              <label class="form-label">Email Address</label>
              <input type="email" id="reg-email" class="form-input" placeholder="sarah@enterprise.com" required>
            </div>

            <div class="form-group">
              <label class="form-label">Role</label>
              <select id="reg-role" class="form-select">
                <option value="cashier">Cashier</option>
                <option value="manager">Manager</option>
                <option value="admin">Administrator</option>
              </select>
            </div>

            <div class="form-group">
              <label class="form-label">Password</label>
              <input type="password" id="reg-password" class="form-input" placeholder="••••••••" required minlength="6">
            </div>

            <button type="submit" class="btn btn-primary" style="width: 100%; margin-top: 8px; padding: 12px;">Create Account</button>
          </form>

          <div style="margin-top: 20px; text-align: center; font-size: 0.875rem; color: var(--text-muted);">
            Already have an account? <a href="#/login">Sign in</a>
          </div>
        </div>
      </div>
    `;

    const form = container.querySelector('#register-form');
    form.onsubmit = async (e) => {
      e.preventDefault();
      const fullName = container.querySelector('#reg-name').value.trim();
      const email = container.querySelector('#reg-email').value.trim();
      const role = container.querySelector('#reg-role').value;
      const password = container.querySelector('#reg-password').value;

      try {
        await Actions.register(email, fullName, password, role);
        window.location.hash = '#/login';
      } catch (err) {
        // Handled in Actions
      }
    };
  }
}
