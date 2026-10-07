// Application Configuration & URL Helpers

const loc = window.location;
const isHttps = loc.protocol === 'https:';
const wsProtocol = isHttps ? 'wss:' : 'ws:';

export const CONFIG = {
  API_BASE: `${loc.protocol}//${loc.host}/api/v1`,
  WS_BASE: `${wsProtocol}//${loc.host}/ws`,
  STORAGE_KEY_TOKEN: 'enterprise_sales_jwt_token',
  STORAGE_KEY_USER: 'enterprise_sales_user_profile',
  WS_RECONNECT_INTERVAL_MS: 3000,
  WS_MAX_RECONNECT_ATTEMPTS: 20,
};
