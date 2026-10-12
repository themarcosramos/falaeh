// Fala Eh - Service Worker
// Cache básico de assets essenciais e fallback offline simples para PWA

const CACHE_NAME = "falaeh-v5";
const STATIC_ASSETS = [
    "/",
    "/index.html",
    "/manifest.json",
    "/assets/css/app.css",
    "/assets/js/app.js",
    "/assets/icons/icon.svg",
    "https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css",
];

self.addEventListener("install", (event) => {
    event.waitUntil(
        caches
            .open(CACHE_NAME)
            .then((cache) => {
                return cache.addAll(STATIC_ASSETS);
            })
            .then(() => self.skipWaiting()),
    );
});

self.addEventListener("activate", (event) => {
    event.waitUntil(
        caches
            .keys()
            .then((keys) => {
                return Promise.all(
                    keys.map((key) => {
                        if (key !== CACHE_NAME) {
                            return caches.delete(key);
                        }
                    }),
                );
            })
            .then(() => self.clients.claim()),
    );
});

self.addEventListener("fetch", (event) => {
    const { request } = event;

    // Apenas requisições GET
    if (request.method !== "GET") {
        return;
    }

    const url = new URL(request.url);

    // Requisições de API: prioriza rede (sem cache offline complexo)
    if (url.pathname.startsWith("/api/")) {
        event.respondWith(
            fetch(request).catch(() => {
                return new Response(
                    JSON.stringify({
                        error: "Você está sem conexão com a internet.",
                        offline: true,
                    }),
                    {
                        status: 503,
                        headers: { "Content-Type": "application/json" },
                    },
                );
            }),
        );
        return;
    }

    // Para navegação HTML (recarregamento/abertura offline), retorna cache do index.html se a rede falhar
    if (request.mode === "navigate") {
        event.respondWith(
            fetch(request).catch(async () => {
                const cached = (await caches.match("/index.html")) || (await caches.match("/"));
                return (
                    cached ||
                    new Response("<h1>Você está offline</h1><p>Abra o jogo novamente quando tiver conexão.</p>", {
                        status: 503,
                        headers: { "Content-Type": "text/html; charset=utf-8" },
                    })
                );
            }),
        );
        return;
    }

    // Para assets estáticos: Cache-first com fallback para a rede e atualização em background
    event.respondWith(
        caches.match(request).then((cachedResponse) => {
            if (cachedResponse) {
                return cachedResponse;
            }

            return fetch(request).then((networkResponse) => {
                if (networkResponse && networkResponse.status === 200 && networkResponse.type === "basic") {
                    const responseClone = networkResponse.clone();
                    caches.open(CACHE_NAME).then((cache) => {
                        cache.put(request, responseClone);
                    });
                }
                return networkResponse;
            });
        }),
    );
});
