export const environment = {
  API_URL: 'http://localhost:5000/template/api',
  MEDIA_URL: 'http://localhost:5000/template/public',
  SESSION: {
    COUNTDOWN_ENABLED: true,
    WARN_SECONDS: 60,
  },
  MAPS: {
    initialLat: -12.005424268674062,
    initialLng: -77.06810865375328,
    initialZoom: 15.5,
    tileUrl: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
  },
};
