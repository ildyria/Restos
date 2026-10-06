import L from 'leaflet'

/**
 * OpenStreetMap services reject requests without a Referer (403). Set it per request so a
 * page-wide `Referrer-Policy: no-referrer` (e.g. from a reverse proxy) can't strip it; only
 * the origin is sent, never the page path.
 */
export const OSM_REFERRER_POLICY = 'strict-origin-when-cross-origin'

/** OpenStreetMap tiles (no API key); main.css greys them to match the page and inverts them in dark mode. */
export function baseTiles() {
  return L.tileLayer('https://tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    referrerPolicy: OSM_REFERRER_POLICY,
    attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
  })
}
