import { Actions } from '../state/actions.js';

export class LoginView {
  static render(container) {
    container.innerHTML = `
      <div class="auth-wrapper fade-in">
        <div class="auth-card">
          <div class="auth-header">
            <h2>Welcome Back</h2>
            <p style="color: var(--text-muted); font-size: 0.875rem; margin-top: 4px;">Sign in to your enterprise sales portal</p>
          </div>

          <form id="login-form">
            <div class="form-group">
              <label class="form-label">Email Address</label>
              <input type="email" id="login-email" class="form-input" placeholder="admin@enterprise.com" required value="admin@enterprise.com">
            </div>

            <div class="form-group">
              <label class="form-label">Password</label>
              <input type="password" id="login-password" class="form-input" placeholder="••••••••" required value="Admin@123456">
            </div>

            <button type="submit" class="btn btn-primary" style="width: 100%; margin-top: 8px; padding: 12px;">Sign In</button>
          </form>

          <div style="margin-top: 20px; text-align: center; font-size: 0.875rem; color: var(--text-muted);">
            Don't have an account? <a href="#/register">Create one here</a>
          </div>
        </div>
      </div>
    `;

    const form = container.querySelector('#login-form');
    form.onsubmit = async (e) => {
      e.preventDefault();
      const email = container.querySelector('#login-email').value.trim();
      const password = container.querySelector('#login-password').value;

      try {
        await Actions.login(email, password);
        window.location.hash = '#/products';
      } catch (err) {
        // Handled in Actions
      }
    };
  }
}
