const CACHE_NAME = "falaeh-v1";
const ASSETS = [
    "/",
    "/index.html",
    "/manifest.json",
    "/assets/css/app.css",
    "/assets/js/app.js",
    "/assets/icons/icon.svg",
];

self.addEventListener("install", (event) => {
    event.waitUntil(caches.open(CACHE_NAME).then((cache) => cache.addAll(ASSETS)));
});

self.addEventListener("activate", (event) => {
    event.waitUntil(
        caches
            .keys()
            .then((keys) => Promise.all(keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key)))),
    );
});

self.addEventListener("fetch", (event) => {
    const url = new URL(event.request.url);

    if (event.request.method !== "GET" || url.origin !== self.location.origin || url.pathname.startsWith("/api/")) {
        return;
    }

    event.respondWith(caches.match(event.request).then((cached) => cached ?? fetch(event.request)));
});
