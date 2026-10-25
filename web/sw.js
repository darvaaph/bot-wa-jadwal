// sw.js — Service Worker untuk PWA Bot Jadwal
const CACHE_NAME = 'bot-jadwal-pwa-v1';

const STATIC_PRECACHE = [
  '/c/',
  '/manifest.webmanifest',
  '/assets/icons/favicon.svg',
  '/assets/icons/icon-192.png',
  '/assets/icons/icon-512.png',
  '/assets/icons/icon-maskable-192.png',
  '/assets/icons/icon-maskable-512.png',
  '/assets/icons/apple-touch-icon.png',
  '/css/tailwind.css',
  '/css/fonts.css',
  '/css/style.css',
  '/css/tokens.css',
  '/css/components.css',
  '/js/vendor/alpine.cdn.min.js',
  '/js/theme-init.js',
  '/js/api.js',
  '/js/shell.js',
  '/js/app-portal.js',
  '/partials/portal/gate.html',
  '/partials/portal/view-dashboard.html',
  '/partials/portal/view-jadwal.html',
  '/partials/portal/view-tugas.html',
  '/partials/portal/view-materi.html',
  '/partials/common/shell.html'
];

// 1. Install Event: Pre-cache aset shell esensial
self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => {
      return cache.addAll(STATIC_PRECACHE).catch((err) => {
        console.debug('Beberapa aset gagal di-precache:', err);
      });
    })
  );
  self.skipWaiting();
});

// 2. Activate Event: Bersihkan cache versi lama
self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) => {
      return Promise.all(
        keys.map((key) => {
          if (key !== CACHE_NAME) {
            return caches.delete(key);
          }
        })
      );
    }).then(() => self.clients.claim())
  );
});

// 3. Message Event: Dukungan skipWaiting dari client jika ada pembaruan
self.addEventListener('message', (event) => {
  if (event.data && (event.data.type === 'SKIP_WAITING' || event.data.action === 'skipWaiting')) {
    self.skipWaiting();
  }
});

// 4. Fetch Event: Strategi Caching
self.addEventListener('fetch', (event) => {
  const req = event.request;
  if (req.method !== 'GET') return;

  const url = new URL(req.url);

  // Jangan tangani resource lintas domain (eksternal)
  if (url.origin !== self.location.origin) return;

  // A. Halaman Navigasi HTML (/c/, /c/:slug, dll): Network-First
  if (req.mode === 'navigate' || url.pathname.startsWith('/c')) {
    event.respondWith(
      fetch(req)
        .then((networkRes) => {
          if (networkRes.status === 200) {
            const resClone = networkRes.clone();
            caches.open(CACHE_NAME).then((cache) => cache.put(req, resClone));
          }
          return networkRes;
        })
        .catch(() => {
          // Jika offline, fallback ke cache request ini atau halaman /c/
          return caches.match(req).then((cached) => {
            return cached || caches.match('/c/');
          });
        })
    );
    return;
  }

  // B. API Data Jadwal & Tugas (/api/v1/portal/): Network-First dengan Cache Fallback
  if (url.pathname.startsWith('/api/v1/portal/')) {
    event.respondWith(
      fetch(req)
        .then((networkRes) => {
          if (networkRes.status === 200) {
            const resClone = networkRes.clone();
            caches.open(CACHE_NAME).then((cache) => cache.put(req, resClone));
          }
          return networkRes;
        })
        .catch(() => caches.match(req))
    );
    return;
  }

  // C. Aset Statis (CSS, JS, Fonts, Images/Icons, Partials): Stale-While-Revalidate
  if (
    url.pathname.startsWith('/css/') ||
    url.pathname.startsWith('/js/') ||
    url.pathname.startsWith('/assets/') ||
    url.pathname.startsWith('/partials/')
  ) {
    event.respondWith(
      caches.match(req).then((cachedRes) => {
        const fetchPromise = fetch(req)
          .then((networkRes) => {
            if (networkRes.status === 200) {
              const resClone = networkRes.clone();
              caches.open(CACHE_NAME).then((cache) => cache.put(req, resClone));
            }
            return networkRes;
          })
          .catch(() => cachedRes);

        return cachedRes || fetchPromise;
      })
    );
    return;
  }

  // D. Default: Fetch dengan fallback cache biasa
  event.respondWith(
    caches.match(req).then((cached) => cached || fetch(req))
  );
});
