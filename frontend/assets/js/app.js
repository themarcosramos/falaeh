// Fala Eh - Jogo Educativo de Exercícios Fonoaudiológicos
// Frontend moderno sem framework SPA (Vanilla JS + Fetch API + Web Speech API + PWA)

const state = {
    sessionId: null,
    currentLevel: "beginner",
    currentLevelName: "Planeta Sons",
    unlockedLevels: ["beginner"],
    exercise: null,
    nextExercise: null,
    exerciseIndex: 0,
    totalExercises: 0,
    selectedOption: null,
    isSubmitting: false,
    isAnsweredCorrectly: false,
    isRecording: false,
    phaseCompleted: false,
    report: null,
    completion: null,
    xp: 0,
    streak: 0,
};

// Mapeamento dos Mundos
const WORLDS = {
    beginner: {
        id: "beginner",
        name: "Planeta Sons",
        icon: "🌱",
        next: "intermediate",
        badge: "✨ NÍVEL 1",
    },
    intermediate: {
        id: "intermediate",
        name: "Vale das Palavras",
        icon: "🌊",
        next: "advanced",
        badge: "✨ NÍVEL 2",
    },
    advanced: {
        id: "advanced",
        name: "Galáxia da Fala",
        icon: "🪐",
        next: null,
        badge: "✨ NÍVEL 3",
    },
};

// Suporte à Web Speech API
const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
const isSpeechRecognitionSupported = Boolean(SpeechRecognition);
let recognitionInstance = null;

// Elementos do DOM
const statusEl = document.getElementById("api-status");
const toastEl = document.getElementById("toast-message");
const toastIconEl = document.getElementById("toast-icon");
const toastTextEl = document.getElementById("toast-text");

const screenHome = document.getElementById("screen-home");
const screenGame = document.getElementById("screen-game");
const screenCelebration = document.getElementById("screen-celebration");

const hudWorldName = document.getElementById("hud-world-name");
const hudXp = document.getElementById("hud-xp");
const hudStreak = document.getElementById("hud-streak");
const hudProgressText = document.getElementById("hud-progress-text");
const hudProgressPercent = document.getElementById("hud-progress-percent");
const hudProgressBar = document.getElementById("hud-progress-bar");

const gameLoading = document.getElementById("game-loading");
const gameError = document.getElementById("game-error");
const exerciseCard = document.getElementById("exercise-card");

const exerciseTypeBadge = document.getElementById("exercise-type-badge");
const exerciseStepIndicator = document.getElementById("exercise-step-indicator");
const exerciseInstruction = document.getElementById("exercise-instruction");
const targetWordContainer = document.getElementById("target-word-container");
const exerciseTargetWord = document.getElementById("exercise-target-word");
const optionsGrid = document.getElementById("options-grid");

// Componentes de Voz
const voiceInteractionArea = document.getElementById("voice-interaction-area");
const btnVoiceRecord = document.getElementById("btn-voice-record");
const voiceStatusBadge = document.getElementById("voice-status-badge");
const voiceStatusIcon = document.getElementById("voice-status-icon");
const voiceStatusText = document.getElementById("voice-status-text");
const voiceTranscriptContainer = document.getElementById("voice-transcript-container");
const voiceTranscriptText = document.getElementById("voice-transcript-text");
const voiceUnsupportedMsg = document.getElementById("voice-unsupported-msg");
const voiceFallbackLabel = document.getElementById("voice-fallback-label");
const voiceDisclaimer = document.getElementById("voice-disclaimer");

const feedbackMessageContainer = document.getElementById("feedback-message-container");
const feedbackBanner = document.getElementById("feedback-banner");
const feedbackIcon = document.getElementById("feedback-icon");
const feedbackText = document.getElementById("feedback-text");

const btnSubmitAnswer = document.getElementById("btn-submit-answer");
const btnNextExercise = document.getElementById("btn-next-exercise");
const btnBackHome = document.getElementById("btn-back-home");
const btnRetryLoad = document.getElementById("btn-retry-load");

const btnStartAdventure = document.getElementById("btn-start-adventure");
const btnSelectBeginner = document.getElementById("btn-select-beginner");
const btnSelectIntermediate = document.getElementById("btn-select-intermediate");
const btnSelectAdvanced = document.getElementById("btn-select-advanced");

const btnReplayWorld = document.getElementById("btn-replay-world");
const btnNextWorld = document.getElementById("btn-next-world");
const btnInstallPwa = document.getElementById("btn-install-pwa");

let deferredInstallPrompt = null;
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
    }, 3500);
}

function showScreen(screen) {
    stopVoiceRecognition();

    [screenHome, screenGame, screenCelebration].forEach((s) => {
        if (s) s.classList.add("d-none");
    });
    if (screen) {
        screen.classList.remove("d-none");
        window.scrollTo({ top: 0, behavior: "smooth" });
    }
}

function normalizeSpeechText(text) {
    if (!text) return "";
    return text
        .trim()
        .toLowerCase()
        .replace(/[.,/#!$%^&*;:{}=\-_`~()?"'!\\]/g, "")
        .replace(/\s+/g, " ");
}

function getTypeMeta(type) {
    switch (type) {
        case "multiple_choice":
            return { label: "🎯 Múltipla Escolha", icon: "🎯" };
        case "image_word_match":
            return { label: "🖼️ Associação de Imagem", icon: "🖼️" };
        case "sound_identification":
            return { label: "🔊 Identificação de Som", icon: "🔊" };
        case "voice":
            return { label: "🗣️ Desafio da Fala", icon: "🗣️" };
        default:
            return { label: "⭐ Desafio Sonoro", icon: "⭐" };
    }
}

function updateWorldCardsUI() {
    Object.keys(WORLDS).forEach((levelId) => {
        const isUnlocked = state.unlockedLevels.includes(levelId);
        const card = document.getElementById(`card-${levelId}`);
        const btn = document.getElementById(`btn-select-${levelId}`);
        const statusBadge = document.getElementById(`badge-status-${levelId}`);
        const pillBadge = document.getElementById(`badge-pill-${levelId}`);

        if (card) {
            if (isUnlocked) {
                card.classList.remove("locked");
                card.classList.add("unlocked");
                card.setAttribute("tabindex", "0");
                card.setAttribute("aria-label", `${WORLDS[levelId].name}, liberado`);
            } else {
                card.classList.add("locked");
                card.classList.remove("unlocked");
                card.setAttribute("aria-label", `${WORLDS[levelId].name}, bloqueado`);
            }
        }

        if (statusBadge && isUnlocked) {
            statusBadge.className = "badge-status-unlocked";
            statusBadge.textContent = WORLDS[levelId].badge;
        }

        if (pillBadge && isUnlocked) {
            pillBadge.className = "badge text-bg-success px-2 py-1";
            pillBadge.textContent = "Disponível";
        }

        if (btn) {
            if (isUnlocked) {
                btn.className = "btn btn-world btn-world-unlocked";
                btn.innerHTML = `<span>Explorar Mundo</span> <span aria-hidden="true">➔</span>`;
            } else {
                btn.className = "btn btn-world btn-world-locked";
                btn.innerHTML = `<span aria-hidden="true">🔒</span> <span>Bloqueado</span>`;
            }
        }
    });
}

// Requisições à API REST
async function apiPost(path, body) {
    const response = await fetch(path, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
    });

    let payload = null;
    try {
        payload = await response.json();
    } catch {
        payload = null;
    }

    if (!response.ok) {
        const error = new Error(payload?.error || `Falha na requisição (${response.status})`);
        error.status = response.status;
        throw error;
    }

    return payload;
}

function setGameLoading(isLoading, hasError = false) {
    if (gameLoading) gameLoading.classList.toggle("d-none", !isLoading);
    if (gameError) gameError.classList.toggle("d-none", !hasError);
    if (exerciseCard) exerciseCard.classList.toggle("d-none", isLoading || hasError);
}

function updateHudStats() {
    if (hudXp) hudXp.textContent = state.xp;
    if (hudStreak) hudStreak.textContent = state.streak;
}

function applyProgress(progress) {
    if (!progress) return;

    state.xp = progress.totalXp ?? state.xp;
    state.streak = progress.currentStreak ?? state.streak;

    if (Array.isArray(progress.unlockedLevels) && progress.unlockedLevels.length > 0) {
        state.unlockedLevels = progress.unlockedLevels;
        updateWorldCardsUI();
    }

    updateHudStats();
}

function announceAchievements(achievements) {
    if (!Array.isArray(achievements) || achievements.length === 0) return;

    const achievement = achievements[achievements.length - 1];
    showToast(achievement.icon || "🏅", `Conquista desbloqueada: ${achievement.title}`);
}

async function startWorld(levelId) {
    if (!state.unlockedLevels.includes(levelId)) {
        showToast("🔒", `Conclua o mundo anterior para desbloquear este nível!`);
        return;
    }

    state.currentLevel = levelId;
    state.currentLevelName = WORLDS[levelId]?.name || "Mundo";
    state.selectedOption = null;
    state.isAnsweredCorrectly = false;
    state.phaseCompleted = false;
    state.nextExercise = null;
    state.report = null;
    state.completion = null;

    showScreen(screenGame);

    if (hudWorldName) {
        hudWorldName.textContent = `${WORLDS[levelId]?.icon || "🌍"} ${state.currentLevelName}`;
    }

    setGameLoading(true);

    try {
        const data = await apiPost("/api/v1/game/start", {
            level: levelId,
            sessionId: state.sessionId || "",
        });

        state.sessionId = data.sessionId;
        state.exercise = data.exercise;
        state.exerciseIndex = data.exerciseIndex ?? 0;
        state.totalExercises = data.totalExercises ?? 0;
        applyProgress(data.progress);

        setGameLoading(false);
        renderCurrentExercise();
    } catch (err) {
        console.error("Falha ao iniciar a partida:", err);

        if (err.status === 403) {
            setGameLoading(false);
            showScreen(screenHome);
            showToast("🔒", "Conclua o mundo anterior para desbloquear este nível!");
            return;
        }

        setGameLoading(false, true);
    }
}

// Configuração e ciclo de vida do Reconhecimento de Voz (Web Speech API)
function setupVoiceRecognition() {
    if (!isSpeechRecognitionSupported) return null;

    try {
        const recognition = new SpeechRecognition();
        recognition.lang = "pt-BR";
        recognition.continuous = false;
        recognition.interimResults = false;
        recognition.maxAlternatives = 1;

        recognition.onstart = () => {
            state.isRecording = true;
            if (btnVoiceRecord) btnVoiceRecord.classList.add("listening");
            if (voiceStatusBadge) {
                voiceStatusBadge.classList.add("listening");
                if (voiceStatusIcon) voiceStatusIcon.textContent = "🔴";
                if (voiceStatusText) voiceStatusText.textContent = "Ouvindo... Fale agora!";
            }
            if (voiceTranscriptContainer) voiceTranscriptContainer.classList.add("d-none");
        };

        recognition.onresult = (event) => {
            state.isRecording = false;
            resetVoiceUIState();

            if (!event.results || !event.results[0] || !event.results[0][0]) return;

            const rawTranscript = event.results[0][0].transcript;

            if (voiceTranscriptContainer && voiceTranscriptText) {
                voiceTranscriptText.textContent = `"${rawTranscript}"`;
                voiceTranscriptContainer.classList.remove("d-none");
            }

            // Submete o texto reconhecido normalizado para a API (sem gravar áudio)
            submitVoiceAnswer(rawTranscript);
        };

        recognition.onerror = (event) => {
            state.isRecording = false;
            resetVoiceUIState();

            let friendlyMessage = "Não conseguimos ouvir com clareza. Tente novamente ou escolha uma opção abaixo.";
            if (event.error === "no-speech") {
                friendlyMessage = "Nenhuma fala detectada. Toque no microfone e tente falar mais perto.";
            } else if (event.error === "not-allowed" || event.error === "permission-denied") {
                friendlyMessage = "Permissão de microfone não concedida. Você pode responder tocando nas opções abaixo!";
                if (voiceUnsupportedMsg) {
                    voiceUnsupportedMsg.textContent = "ℹ️ Microfone indisponível ou permissão não concedida. Você pode responder tocando nas opções abaixo!";
                    voiceUnsupportedMsg.classList.remove("d-none");
                }
            } else if (event.error === "network") {
                friendlyMessage = "Falha temporária de rede no reconhecimento. Tente novamente ou use as opções de toque.";
            }

            showToast("🎙️", friendlyMessage);
            if (voiceStatusText) voiceStatusText.textContent = "Toque no microfone para tentar de novo";
        };

        recognition.onend = () => {
            state.isRecording = false;
            resetVoiceUIState();
        };

        return recognition;
    } catch (e) {
        console.warn("Erro ao configurar SpeechRecognition:", e);
        return null;
    }
}

function resetVoiceUIState() {
    if (btnVoiceRecord) btnVoiceRecord.classList.remove("listening");
    if (voiceStatusBadge) {
        voiceStatusBadge.classList.remove("listening");
        if (voiceStatusIcon) voiceStatusIcon.textContent = "🎙️";
        if (voiceStatusText) voiceStatusText.textContent = "Toque no microfone para falar";
    }
}

function stopVoiceRecognition() {
    if (recognitionInstance && state.isRecording) {
        try {
            recognitionInstance.stop();
        } catch {}
    }
    state.isRecording = false;
    resetVoiceUIState();
}

function toggleVoiceRecognition() {
    if (state.isAnsweredCorrectly || state.isSubmitting) return;

    if (!isSpeechRecognitionSupported) {
        if (voiceUnsupportedMsg) voiceUnsupportedMsg.classList.remove("d-none");
        showToast("ℹ️", "Reconhecimento de voz não suportado neste navegador. Escolha uma das opções por toque!");
        return;
    }

    if (!recognitionInstance) {
        recognitionInstance = setupVoiceRecognition();
    }

    if (!recognitionInstance) {
        if (voiceUnsupportedMsg) voiceUnsupportedMsg.classList.remove("d-none");
        return;
    }

    if (state.isRecording) {
        try {
            recognitionInstance.stop();
        } catch {}
        state.isRecording = false;
        resetVoiceUIState();
    } else {
        try {
            recognitionInstance.start();
        } catch (err) {
            console.warn("Aviso ao iniciar reconhecimento de fala:", err);
            try {
                recognitionInstance.stop();
                setTimeout(() => recognitionInstance.start(), 200);
            } catch {}
        }
    }
}

async function submitVoiceAnswer(spokenText) {
    if (!spokenText || state.isSubmitting || state.isAnsweredCorrectly) return;

    const currentEx = state.exercise;
    if (!currentEx || !state.sessionId) return;

    state.selectedOption = spokenText;
    state.isSubmitting = true;

    if (btnSubmitAnswer) {
        btnSubmitAnswer.disabled = true;
        btnSubmitAnswer.textContent = "Verificando…";
    }

    // Se o texto dito corresponder a uma das opções, destaca a respectiva opção visualmente
    const normalizedSpoken = normalizeSpeechText(spokenText);
    const allButtons = optionsGrid.querySelectorAll(".option-btn");
    allButtons.forEach((btn) => {
        const optText = btn.querySelector("span:last-child")?.textContent || "";
        if (normalizeSpeechText(optText) === normalizedSpoken) {
            btn.classList.add("selected");
            btn.setAttribute("aria-checked", "true");
        }
    });

    try {
        const data = await apiPost(`/api/v1/game/${encodeURIComponent(state.sessionId)}/answer`, {
            exerciseId: currentEx.id,
            answer: spokenText,
        });

        handleAnswerResult(data);
    } catch (err) {
        console.error("Falha ao submeter resposta por voz:", err);
        showToast("⚠️", "Erro ao conectar com a API. Tente novamente.");
        if (btnSubmitAnswer) {
            btnSubmitAnswer.disabled = false;
            btnSubmitAnswer.textContent = "✨ Confirmar Resposta";
        }
    } finally {
        state.isSubmitting = false;
    }
}

function renderCurrentExercise() {
    const currentEx = state.exercise;
    if (!currentEx) return;

    stopVoiceRecognition();
    state.selectedOption = null;
    state.isAnsweredCorrectly = false;

    // Atualiza HUD
    const total = state.totalExercises || 1;
    const currentNum = Math.min(state.exerciseIndex + 1, total);
    const progressPct = Math.round((currentNum / total) * 100);

    updateHudStats();
    if (hudProgressText) hudProgressText.textContent = `Exercício ${currentNum} de ${total}`;
    if (hudProgressPercent) hudProgressPercent.textContent = `${progressPct}%`;
    if (hudProgressBar) {
        hudProgressBar.style.width = `${progressPct}%`;
        hudProgressBar.setAttribute("aria-valuenow", progressPct);
    }

    // Atualiza cabeçalho do exercício
    const typeMeta = getTypeMeta(currentEx.type);
    if (exerciseTypeBadge) exerciseTypeBadge.textContent = typeMeta.label;
    if (exerciseStepIndicator) exerciseStepIndicator.textContent = `Missão ${currentNum}`;
    if (exerciseInstruction) exerciseInstruction.textContent = currentEx.instruction;

    // Palavra-alvo
    if (targetWordContainer && exerciseTargetWord) {
        if (currentEx.targetWord) {
            exerciseTargetWord.textContent = currentEx.targetWord;
            targetWordContainer.classList.remove("d-none");
        } else {
            targetWordContainer.classList.add("d-none");
        }
    }

    // Tratamento de Exercício por Voz vs Múltipla Escolha
    const isVoiceType = currentEx.type === "voice";

    if (voiceInteractionArea) {
        if (isVoiceType) {
            voiceInteractionArea.classList.remove("d-none");
            resetVoiceUIState();
            if (voiceTranscriptContainer) voiceTranscriptContainer.classList.add("d-none");

            if (!isSpeechRecognitionSupported) {
                if (voiceUnsupportedMsg) voiceUnsupportedMsg.classList.remove("d-none");
                if (btnVoiceRecord) btnVoiceRecord.disabled = true;
                if (voiceStatusText) voiceStatusText.textContent = "Reconhecimento de voz não suportado neste navegador";
            } else {
                if (voiceUnsupportedMsg) voiceUnsupportedMsg.classList.add("d-none");
                if (btnVoiceRecord) btnVoiceRecord.disabled = false;
            }

            if (voiceFallbackLabel) {
                voiceFallbackLabel.classList.remove("d-none");
            }
        } else {
            voiceInteractionArea.classList.add("d-none");
            if (voiceFallbackLabel) voiceFallbackLabel.classList.add("d-none");
        }
    }

    if (voiceDisclaimer) {
        voiceDisclaimer.classList.toggle("d-none", !isVoiceType);
    }

    // Renderiza opções manuais (sempre disponíveis como opção primária ou fallback)
    if (optionsGrid) {
        optionsGrid.innerHTML = "";
        const options = currentEx.options || [];

        options.forEach((optText, index) => {
            const col = document.createElement("div");
            col.className = options.length <= 2 ? "col-12 col-md-6" : "col-12 col-sm-6 col-lg-4";

            const btn = document.createElement("button");
            btn.type = "button";
            btn.className = "option-btn";
            btn.setAttribute("role", "radio");
            btn.setAttribute("aria-checked", "false");
            btn.setAttribute("aria-label", `Opção ${index + 1}: ${optText}`);

            btn.innerHTML = `
                <span class="option-letter badge bg-light text-dark rounded-circle me-2" style="width: 28px; height: 28px; display: inline-flex; align-items: center; justify-content: center; font-size: 0.85rem;">
                    ${String.fromCharCode(65 + index)}
                </span>
                <span>${optText}</span>
            `;

            btn.addEventListener("click", () => {
                if (state.isAnsweredCorrectly || state.isSubmitting) return;
                selectOption(optText, btn);
            });

            col.appendChild(btn);
            optionsGrid.appendChild(col);
        });
    }

    // Reseta painel de feedback e botões de ação
    if (feedbackMessageContainer) feedbackMessageContainer.classList.add("d-none");
    if (btnSubmitAnswer) {
        btnSubmitAnswer.classList.remove("d-none");
        btnSubmitAnswer.disabled = true;
        btnSubmitAnswer.textContent = "✨ Confirmar Resposta";
    }
    if (btnNextExercise) {
        btnNextExercise.classList.add("d-none");
    }
}

function selectOption(optText, clickedBtn) {
    state.selectedOption = optText;

    const allButtons = optionsGrid.querySelectorAll(".option-btn");
    allButtons.forEach((btn) => {
        btn.classList.remove("selected");
        btn.setAttribute("aria-checked", "false");
    });

    clickedBtn.classList.add("selected");
    clickedBtn.setAttribute("aria-checked", "true");

    if (btnSubmitAnswer) {
        btnSubmitAnswer.disabled = false;
    }
}

async function submitAnswer() {
    if (!state.selectedOption || state.isSubmitting || state.isAnsweredCorrectly) return;

    const currentEx = state.exercise;
    if (!currentEx || !state.sessionId) return;

    stopVoiceRecognition();
    state.isSubmitting = true;

    if (btnSubmitAnswer) {
        btnSubmitAnswer.disabled = true;
        btnSubmitAnswer.textContent = "Verificando…";
    }

    try {
        const data = await apiPost(`/api/v1/game/${encodeURIComponent(state.sessionId)}/answer`, {
            exerciseId: currentEx.id,
            answer: state.selectedOption,
        });

        handleAnswerResult(data);
    } catch (err) {
        console.error("Falha ao submeter resposta:", err);
        showToast("⚠️", "Erro ao conectar com a API. Tente novamente.");
        if (btnSubmitAnswer) {
            btnSubmitAnswer.disabled = false;
            btnSubmitAnswer.textContent = "✨ Confirmar Resposta";
        }
    } finally {
        state.isSubmitting = false;
    }
}

function handleAnswerResult(data) {
    const isCorrect = Boolean(data.correct);
    const allButtons = optionsGrid.querySelectorAll(".option-btn");
    const selectedBtn = Array.from(allButtons).find((b) => b.classList.contains("selected"));

    applyProgress(data.progress);

    if (isCorrect) {
        state.isAnsweredCorrectly = true;
        state.exerciseIndex = data.exerciseIndex ?? state.exerciseIndex;
        state.totalExercises = data.totalExercises || state.totalExercises;
        state.nextExercise = data.nextExercise || null;
        state.phaseCompleted = Boolean(data.phaseCompleted);
        state.report = data.report || null;
        state.completion = data.completion || null;

        // Visual do botão correto
        if (selectedBtn) {
            selectedBtn.classList.remove("selected");
            selectedBtn.classList.add("correct");
        }

        // Desabilita botões para evitar cliques extras
        allButtons.forEach((b) => (b.disabled = true));
        if (btnVoiceRecord) btnVoiceRecord.disabled = true;

        // Feedback positivo
        if (feedbackMessageContainer && feedbackBanner && feedbackIcon && feedbackText) {
            feedbackBanner.className = "d-inline-flex align-items-center gap-2 px-4 py-2 rounded-pill fw-bold feedback-banner-correct";
            feedbackIcon.textContent = "🎉";
            feedbackText.textContent = `Muito bem! Resposta correta! (+${data.earnedXp ?? 0} XP)`;
            feedbackMessageContainer.classList.remove("d-none");
        }

        announceAchievements(data.newAchievements);

        // Transiciona botão de ação para "Próximo"
        if (btnSubmitAnswer) btnSubmitAnswer.classList.add("d-none");
        if (btnNextExercise) {
            btnNextExercise.textContent = state.phaseCompleted ? "🏆 Ver Resultado" : "Próximo desafio ➔";
            btnNextExercise.classList.remove("d-none");
            btnNextExercise.focus();
        }
    } else {
        if (selectedBtn) {
            selectedBtn.classList.remove("selected");
            selectedBtn.classList.add("incorrect");
            selectedBtn.classList.add("shake");
            setTimeout(() => selectedBtn.classList.remove("shake"), 500);
        }

        // Feedback amigável de erro
        if (feedbackMessageContainer && feedbackBanner && feedbackIcon && feedbackText) {
            feedbackBanner.className = "d-inline-flex align-items-center gap-2 px-4 py-2 rounded-pill fw-bold feedback-banner-incorrect";
            feedbackIcon.textContent = "💡";
            feedbackText.textContent = "Quase lá! Tente novamente ou escolha outra opção.";
            feedbackMessageContainer.classList.remove("d-none");
        }

        if (btnSubmitAnswer) {
            btnSubmitAnswer.disabled = true;
            btnSubmitAnswer.textContent = "✨ Confirmar Resposta";
        }
    }
}

function goToNextExercise() {
    if (state.phaseCompleted || !state.nextExercise) {
        finishWorld();
        return;
    }

    state.exercise = state.nextExercise;
    state.nextExercise = null;
    renderCurrentExercise();
}

function finishWorld() {
    stopVoiceRecognition();

    const currentWorld = WORLDS[state.currentLevel];
    const report = state.report;
    const nextWorldId = state.completion?.nextLevel || null;
    const unlockedNext = Boolean(state.completion?.nextUnlocked);

    // Preenche tela de celebração
    const celebrationWorldName = document.getElementById("celebration-world-name");
    const celebrationSubtitle = document.getElementById("celebration-subtitle");
    const summaryXp = document.getElementById("summary-xp");
    const summaryAccuracy = document.getElementById("summary-accuracy");
    const summaryStreak = document.getElementById("summary-streak");

    if (celebrationWorldName) celebrationWorldName.textContent = currentWorld?.name || "Mundo";
    if (celebrationSubtitle) {
        celebrationSubtitle.textContent = unlockedNext
            ? `Incrível! Você concluiu o ${currentWorld?.name} e desbloqueou o próximo nível!`
            : `Parabéns! Você concluiu todos os desafios do ${currentWorld?.name}!`;
    }

    if (summaryXp) summaryXp.textContent = `${report?.totalXp ?? state.xp} XP`;
    if (summaryAccuracy) summaryAccuracy.textContent = `${report?.accuracy ?? 0}%`;
    if (summaryStreak) summaryStreak.textContent = `${report?.bestStreak ?? 0}`;

    if (btnNextWorld) {
        if (unlockedNext && nextWorldId && WORLDS[nextWorldId]) {
            btnNextWorld.textContent = `🚀 Próximo Mundo (${WORLDS[nextWorldId].name})`;
            btnNextWorld.onclick = () => startWorld(nextWorldId);
        } else {
            btnNextWorld.textContent = "🗺️ Escolher Outro Mundo";
            btnNextWorld.onclick = () => showScreen(screenHome);
        }
    }

    showScreen(screenCelebration);
}

function initEventHandlers() {
    if (btnStartAdventure) {
        btnStartAdventure.addEventListener("click", () => startWorld("beginner"));
    }

    if (btnSelectBeginner) {
        btnSelectBeginner.addEventListener("click", () => startWorld("beginner"));
    }

    if (btnSelectIntermediate) {
        btnSelectIntermediate.addEventListener("click", () => startWorld("intermediate"));
    }

    if (btnSelectAdvanced) {
        btnSelectAdvanced.addEventListener("click", () => startWorld("advanced"));
    }

    if (btnVoiceRecord) {
        btnVoiceRecord.addEventListener("click", toggleVoiceRecognition);
    }

    if (btnSubmitAnswer) {
        btnSubmitAnswer.addEventListener("click", submitAnswer);
    }

    if (btnNextExercise) {
        btnNextExercise.addEventListener("click", goToNextExercise);
    }

    if (btnBackHome) {
        btnBackHome.addEventListener("click", () => showScreen(screenHome));
    }

    if (btnRetryLoad) {
        btnRetryLoad.addEventListener("click", () => startWorld(state.currentLevel));
    }

    if (btnReplayWorld) {
        btnReplayWorld.addEventListener("click", () => startWorld(state.currentLevel));
    }

    // Feedback para cards bloqueados
    const lockedCards = document.querySelectorAll(".world-card");
    lockedCards.forEach((card) => {
        card.addEventListener("click", () => {
            const worldId = card.getAttribute("data-world");
            if (worldId && !state.unlockedLevels.includes(worldId)) {
                const worldName = WORLDS[worldId]?.name || "este nível";
                card.classList.remove("shake");
                void card.offsetWidth;
                card.classList.add("shake");
                showToast("🔒", `Conclua o mundo anterior para desbloquear o ${worldName}!`);
            }
        });
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

function initPWA() {
    // Captura e gerenciamento do prompt de instalação do PWA
    window.addEventListener("beforeinstallprompt", (e) => {
        e.preventDefault();
        deferredInstallPrompt = e;
        if (btnInstallPwa) {
            btnInstallPwa.classList.remove("d-none");
            btnInstallPwa.classList.add("d-inline-flex");
        }
    });

    if (btnInstallPwa) {
        btnInstallPwa.addEventListener("click", async () => {
            if (!deferredInstallPrompt) return;
            deferredInstallPrompt.prompt();
            const { outcome } = await deferredInstallPrompt.userChoice;
            if (outcome === "accepted") {
                showToast("🎉", "Obrigado por instalar o Fala Eh!");
            }
            deferredInstallPrompt = null;
            btnInstallPwa.classList.add("d-none");
            btnInstallPwa.classList.remove("d-inline-flex");
        });
    }

    window.addEventListener("appinstalled", () => {
        deferredInstallPrompt = null;
        if (btnInstallPwa) {
            btnInstallPwa.classList.add("d-none");
            btnInstallPwa.classList.remove("d-inline-flex");
        }
        showToast("🌟", "Aplicativo Fala Eh instalado com sucesso!");
    });

    // Detecção de conectividade online/offline simples e amigável
    function handleNetworkStatus() {
        if (!navigator.onLine) {
            if (statusEl) {
                statusEl.textContent = "Modo offline 📴";
                statusEl.className = "badge text-bg-warning text-dark";
            }
            showToast("📴", "Você está offline. A interface do jogo continua disponível!");
        } else {
            checkAPI();
            showToast("📶", "Conexão restabelecida!");
        }
    }

    window.addEventListener("online", handleNetworkStatus);
    window.addEventListener("offline", handleNetworkStatus);
}

if ("serviceWorker" in navigator) {
    window.addEventListener("load", () => {
        navigator.serviceWorker.register("/sw.js").catch(() => {
            /* PWA é progressivo: a aplicação continua funcionando sem service worker */
        });
    });
}

initEventHandlers();
initPWA();
updateWorldCardsUI();
checkAPI();

