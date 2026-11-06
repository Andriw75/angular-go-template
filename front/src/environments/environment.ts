import type { DashboardConfig } from '../app/models/dashboard';

export const environment = {
  APP_NAME: 'Template',
  API_URL: 'http://localhost:5000/template/api',
  MEDIA_URL: 'http://localhost:5000/template/public',
  SESSION: {
    COUNTDOWN_ENABLED: true,
    WARN_SECONDS: 60,
  },
  DASHBOARD: {
    VARIANT: 'sidebar',
    HORIZONTAL: {
      overflow: 'menu-scroll',
      sticky: true,
    },
    GRID: {
      columns: 5,
      showSearch: true,
    },
  } satisfies DashboardConfig,
  MAPS: {
    initialLat: -12.005424268674062,
    initialLng: -77.06810865375328,
    initialZoom: 15.5,
    tileUrl: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
  },
};
