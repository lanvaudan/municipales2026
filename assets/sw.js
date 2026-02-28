// Service Worker PWA Lanvaudan
// Toujours à jour, privilégie le réseau, fallback cache basique

const CACHE_NAME = 'lanvaudan-cache-v2';

self.addEventListener('install', (event) => {
    // Ne pas forcer le cache de tout au démarrage pour éviter les soucis de "pas à jour"
    self.skipWaiting();
});

self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches.keys().then((cacheNames) => {
            return Promise.all(
                cacheNames.map((cacheName) => {
                    if (cacheName !== CACHE_NAME) {
                        return caches.delete(cacheName);
                    }
                })
            );
        })
    );
    return self.clients.claim();
});

self.addEventListener('fetch', (event) => {
    // Stratégie "Network First" : on essaie de récupérer depuis le réseau.
    // Si ça échoue (hors-ligne), on tente de récupérer depuis le cache.
    // Cela garantit que la PWA est "toujours à jour" tant qu'il y a du réseau.
    
    // On ignore les requêtes vers l'API d'abonnement pour ne pas les cacher
    if (event.request.url.includes('/subscribe')) {
        return;
    }

    event.respondWith(
        fetch(event.request)
            .then((response) => {
                // Si la réponse est valide, on clone et on met en cache pour la prochaine fois en hors ligne
                if (response && response.status === 200 && response.type === 'basic') {
                    const responseToCache = response.clone();
                    caches.open(CACHE_NAME)
                        .then((cache) => {
                            cache.put(event.request, responseToCache);
                        });
                }
                return response;
            })
            .catch(() => {
                // Si le fetch échoue, on regarde dans le cache
                return caches.match(event.request);
            })
    );
});
