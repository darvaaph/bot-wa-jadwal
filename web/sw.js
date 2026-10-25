// sw.js — Service Worker untuk PWA Asterisk (Jadwal & Tugas Mahasiswa)
const CACHE_NAME = 'asterisk-pwa-v5';

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
        console.debug('Aset precache parsial:', err);
      });
    })
  );
  self.skipWaiting();
});

// 2. Activate Event: Bersihkan cache versi lama seketika
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

  // A. Halaman Navigasi HTML (/c/, /c/:slug, /, /index.html): Network-First
  if (req.mode === 'navigate' || url.pathname.startsWith('/c') || url.pathname === '/' || url.pathname === '/index.html') {
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
          return caches.match(req).then((cached) => {
            return cached || caches.match('/c/');
          });
        })
    );
    return;
  }

  // B. Seluruh Endpoint API (/api/): Network-First dengan Cache Fallback
  if (url.pathname.startsWith('/api/')) {
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

  // C. Partials HTML & Dynamic App JS: Network-First (agar selalu segar saat online)
  if (
    url.pathname.startsWith('/partials/') ||
    url.pathname === '/js/app-portal.js' ||
    url.pathname === '/js/api.js' ||
    url.pathname === '/js/shell.js' ||
    url.pathname === '/js/theme-init.js'
  ) {
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

  // D. Aset Statis Murni (CSS, Vendor JS, Fonts, Icons): Stale-While-Revalidate
  if (
    url.pathname.startsWith('/css/') ||
    url.pathname.startsWith('/js/vendor/') ||
    url.pathname.startsWith('/assets/')
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

  // E. Default Fallback
  event.respondWith(
    caches.match(req).then((cached) => cached || fetch(req))
  );
});
