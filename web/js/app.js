import { store } from './state/store.js';
import { wsManager } from './ws/socket.js';
import { WSDispatcher } from './ws/dispatcher.js';
import { Navbar } from './components/navbar.js';
import { LoginView } from './views/login_view.js';
import { RegisterView } from './views/register_view.js';
import { ProductsView } from './views/products_view.js';
import { SalesView } from './views/sales_view.js';
import { AuditView } from './views/audit_view.js';

class App {
  static init() {
    // 1. Initialize WebSocket Dispatcher
    WSDispatcher.init();

    // 2. Connect WebSocket if user has stored session
    if (store.isAuthenticated()) {
      wsManager.connect();
    }

    // 3. Listen to state changes for UI updates
    store.on('authChanged', () => {
      App.renderNavigation();
      App.handleRoute();
    });

    store.on('wsStatusChanged', (status) => {
      Navbar.updateWsStatus(status);
    });

    // 4. Listen to browser hash changes
    window.addEventListener('hashchange', () => {
      App.handleRoute();
    });

    // 5. Initial render
    App.renderNavigation();
    App.handleRoute();
  }

  static renderNavigation() {
    const navContainer = document.getElementById('navbar-root');
    if (navContainer) {
      Navbar.render(navContainer);
    }
  }

  static handleRoute() {
    const isAuth = store.isAuthenticated();
    const hash = window.location.hash || (isAuth ? '#/products' : '#/login');
    const mainContainer = document.getElementById('view-root');
    if (!mainContainer) return;

    App.renderNavigation();

    // Route Guard
    if (!isAuth && hash !== '#/login' && hash !== '#/register') {
      window.location.hash = '#/login';
      return;
    }

    if (isAuth && (hash === '#/login' || hash === '#/register')) {
      window.location.hash = '#/products';
      return;
    }

    switch (hash) {
      case '#/login':
        LoginView.render(mainContainer);
        break;

      case '#/register':
        RegisterView.render(mainContainer);
        break;

      case '#/products':
        ProductsView.render(mainContainer);
        break;

      case '#/sales':
        SalesView.render(mainContainer);
        break;

      case '#/audit':
        AuditView.render(mainContainer);
        break;

      default:
        window.location.hash = isAuth ? '#/products' : '#/login';
    }
  }
}

// Bootstrap on DOM ready
document.addEventListener('DOMContentLoaded', () => {
  App.init();
});
