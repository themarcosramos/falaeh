const statusEl = document.getElementById("api-status");
const toastEl = document.getElementById("toast-message");
const toastIconEl = document.getElementById("toast-icon");
const toastTextEl = document.getElementById("toast-text");

let toastTimeoutId = null;

function showToast(icon, message) {
    if (!toastEl || !toastIconEl || !toastTextEl) return;

    toastIconEl.textContent = icon;
    toastTextEl.textContent = message;
    toastEl.classList.add("show");

    if (toastTimeoutId) {
        clearTimeout(toastTimeoutId);
    }

    toastTimeoutId = setTimeout(() => {
        toastEl.classList.remove("show");
    }, 3200);
}

function initHomeScreen() {
    const startBtn = document.getElementById("btn-start-adventure");
    const beginnerBtn = document.getElementById("btn-select-beginner");
    const beginnerCard = document.getElementById("card-beginner");

    const highlightBeginner = () => {
        if (!beginnerCard) return;
        beginnerCard.classList.add("active-focus");
        beginnerCard.scrollIntoView({ behavior: "smooth", block: "center" });
        showToast("🌟", "Mundo 1 (Iniciante) selecionado! Exercícios em breve.");

        setTimeout(() => {
            beginnerCard.classList.remove("active-focus");
        }, 1600);
    };

    if (startBtn) {
        startBtn.addEventListener("click", highlightBeginner);
    }

    if (beginnerBtn) {
        beginnerBtn.addEventListener("click", highlightBeginner);
    }

    if (beginnerCard) {
        beginnerCard.addEventListener("keydown", (e) => {
            if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                highlightBeginner();
            }
        });
    }

    // Feedback para mundos bloqueados
    const lockedButtons = document.querySelectorAll(".world-card.locked");
    lockedButtons.forEach((card) => {
        const handleLockedClick = () => {
            const worldName = card.querySelector("h3")?.textContent || "este nível";
            card.classList.remove("shake");
            // Trigger reflow to restart CSS animation
            void card.offsetWidth;
            card.classList.add("shake");
            showToast("🔒", `Conclua o mundo anterior para desbloquear o ${worldName}!`);
        };

        card.addEventListener("click", handleLockedClick);
    });
}

async function checkAPI() {
    if (!statusEl) return;
    try {
        const response = await fetch("/api/health");
        if (!response.ok) {
            throw new Error(`status ${response.status}`);
        }

        statusEl.textContent = "API online";
        statusEl.className = "badge text-bg-success";
    } catch {
        statusEl.textContent = "API indisponível";
        statusEl.className = "badge text-bg-secondary opacity-75";
    }
}

if ("serviceWorker" in navigator) {
    window.addEventListener("load", () => {
        navigator.serviceWorker.register("/sw.js").catch(() => {
            /* PWA é progressivo: a aplicação continua funcionando sem service worker */
        });
    });
}

initHomeScreen();
checkAPI();

