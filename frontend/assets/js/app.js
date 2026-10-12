const statusEl = document.getElementById("api-status");

async function checkAPI() {
    try {
        const response = await fetch("/api/health");
        if (!response.ok) {
            throw new Error(`status ${response.status}`);
        }

        statusEl.textContent = "API online";
        statusEl.className = "badge text-bg-success";
    } catch {
        statusEl.textContent = "API indisponível";
        statusEl.className = "badge text-bg-danger";
    }
}

if ("serviceWorker" in navigator) {
    window.addEventListener("load", () => {
        navigator.serviceWorker.register("/sw.js").catch(() => {
            /* PWA é progressivo: a aplicação continua funcionando sem service worker */
        });
    });
}

checkAPI();
