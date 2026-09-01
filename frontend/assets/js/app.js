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

const btnExportPdf = document.getElementById("btn-export-pdf");
const btnExportImage = document.getElementById("btn-export-image");

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
        } catch { }
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
        } catch { }
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
            } catch { }
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
                <span>${escapeHtml(optText)}</span>
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

    // Preenche cabeçalho e estatísticas do relatório final
    const celebrationWorldName = document.getElementById("celebration-world-name");
    const celebrationSubtitle = document.getElementById("celebration-subtitle");
    const reportWorldIcon = document.getElementById("report-world-icon");
    const reportLevel = document.getElementById("report-level");
    const reportPhaseCompleted = document.getElementById("report-phase-completed");

    const summaryXp = document.getElementById("summary-xp");
    const summaryAccuracy = document.getElementById("summary-accuracy");
    const summaryStreak = document.getElementById("summary-streak");
    const reportExercisesCount = document.getElementById("report-exercises-count");
    const reportHits = document.getElementById("report-hits");
    const reportAttempts = document.getElementById("report-attempts");
    const reportNextLevel = document.getElementById("report-next-level");

    if (celebrationWorldName) celebrationWorldName.textContent = currentWorld?.name || "Mundo";
    if (celebrationSubtitle) {
        celebrationSubtitle.textContent = unlockedNext
            ? `Incrível! Você concluiu o ${currentWorld?.name} e desbloqueou o próximo nível!`
            : `Parabéns! Você concluiu todos os desafios do ${currentWorld?.name}!`;
    }

    if (reportWorldIcon) reportWorldIcon.textContent = currentWorld?.icon || "🌍";
    if (reportLevel) reportLevel.textContent = `${currentWorld?.name || "Iniciante"} (${currentWorld?.badge || "NÍVEL 1"})`;
    if (reportPhaseCompleted) {
        reportPhaseCompleted.textContent = "Sim 🏆";
        reportPhaseCompleted.className = "badge text-bg-success px-3 py-2 fw-bold";
    }

    const totalXpVal = report?.totalXp ?? state.xp;
    const accuracyVal = report?.accuracy ?? 100;
    const bestStreakVal = report?.bestStreak ?? state.streak;
    const exercisesCountVal = report?.exercisesTotal ?? state.totalExercises;
    const hitsVal = report?.hits ?? state.totalExercises;
    const attemptsVal = report?.attempts ?? state.totalExercises;

    if (summaryXp) summaryXp.textContent = `${totalXpVal} XP`;
    if (summaryAccuracy) summaryAccuracy.textContent = `${accuracyVal}%`;
    if (summaryStreak) summaryStreak.textContent = `${bestStreakVal}`;
    if (reportExercisesCount) reportExercisesCount.textContent = `${exercisesCountVal}`;
    if (reportHits) reportHits.textContent = `${hitsVal}`;
    if (reportAttempts) reportAttempts.textContent = `${attemptsVal}`;

    if (reportNextLevel) {
        if (unlockedNext && nextWorldId && WORLDS[nextWorldId]) {
            reportNextLevel.textContent = `${WORLDS[nextWorldId].name} (${WORLDS[nextWorldId].badge})`;
            reportNextLevel.className = "badge text-bg-primary px-2 py-1 fw-bold";
        } else if (!nextWorldId) {
            reportNextLevel.textContent = "Mestre da Fala! 🎖️";
            reportNextLevel.className = "badge text-bg-success px-2 py-1 fw-bold";
        } else {
            reportNextLevel.textContent = `${WORLDS[nextWorldId]?.name || "Próximo"}`;
            reportNextLevel.className = "badge text-bg-secondary px-2 py-1 fw-bold";
        }
    }

    const reportPlayerName = document.getElementById("report-player-name");
    const reportPlayerNameDisplay = document.getElementById("report-player-name-display");

    if (reportPlayerName && reportPlayerNameDisplay) {
        reportPlayerName.oninput = () => {
            const val = reportPlayerName.value.trim();
            reportPlayerNameDisplay.textContent = val ? `⭐ ${val} ⭐` : "⭐ Astronauta da Fala ⭐";
        };
    }

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

// Nome opcional digitado pelo jogador; permanece apenas no navegador.
function getReportPlayerName() {
    const input = document.getElementById("report-player-name");
    return (input?.value || "").trim() || "Astronauta da Fala";
}

function escapeHtml(value) {
    return String(value).replace(/[&<>"']/g, (char) => {
        switch (char) {
            case "&": return "&amp;";
            case "<": return "&lt;";
            case ">": return "&gt;";
            case "\"": return "&quot;";
            default: return "&#39;";
        }
    });
}

// Helper universal para desenhar retângulos arredondados compatível com todos os navegadores
function drawRoundRect(ctx, x, y, width, height, radius) {
    // beginPath é obrigatório: ctx.roundRect apenas acrescenta ao path atual.
    ctx.beginPath();
    if (typeof ctx.roundRect === "function") {
        ctx.roundRect(x, y, width, height, radius);
        return;
    }
    ctx.moveTo(x + radius, y);
    ctx.lineTo(x + width - radius, y);
    ctx.quadraticCurveTo(x + width, y, x + width, y + radius);
    ctx.lineTo(x + width, y + height - radius);
    ctx.quadraticCurveTo(x + width, y + height, x + width - radius, y + height);
    ctx.lineTo(x + radius, y + height);
    ctx.quadraticCurveTo(x, y + height, x, y + height - radius);
    ctx.lineTo(x, y + radius);
    ctx.quadraticCurveTo(x, y, x + radius, y);
    ctx.closePath();
}

// Geração local de PDF através de documento limpo e impressão estilizada
function exportReportPDF() {
    try {
        showToast("📄", "Preparando relatório em PDF...");
        const currentWorld = WORLDS[state.currentLevel] || { name: "Iniciante (Planeta Sons)", badge: "NÍVEL 1", icon: "🌱" };
        const report = state.report;
        const totalXp = report?.totalXp ?? state.xp ?? 0;
        const accuracy = report?.accuracy ?? 100;
        const bestStreak = report?.bestStreak ?? state.streak ?? 0;
        const exercisesCompleted = report?.exercisesTotal ?? state.totalExercises ?? 0;
        const hits = report?.hits ?? state.totalExercises ?? 0;
        const attempts = report?.attempts ?? state.totalExercises ?? 0;
        const nextLevelName = state.completion?.nextLevel && WORLDS[state.completion.nextLevel]
            ? WORLDS[state.completion.nextLevel].name
            : "Missão Final Concluída! 🎖️";
        const dataStr = new Date().toLocaleDateString("pt-BR", {
            day: "2-digit",
            month: "2-digit",
            year: "numeric",
            hour: "2-digit",
            minute: "2-digit",
        });

        const playerName = getReportPlayerName();

        // Abre visualização de impressão formatada em A4
        const printWin = window.open("", "_blank", "width=850,height=1000");
        if (printWin) {
            printWin.document.open();
            printWin.document.write(`
<!DOCTYPE html>
<html lang="pt-BR">
<head>
    <meta charset="utf-8" />
    <title>Fala Eh - Relatório de Missão</title>
    <style>
        @page { size: A4 portrait; margin: 1.5cm; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; color: #0f172a; margin: 0; padding: 24px; background: #ffffff; }
        .header { text-align: center; border-bottom: 2px solid #8b5cf6; padding-bottom: 16px; margin-bottom: 20px; }
        .logo { font-size: 32px; font-weight: 800; color: #4c1d95; }
        .tagline { font-size: 14px; color: #64748b; text-transform: uppercase; letter-spacing: 1.5px; font-weight: 600; margin-top: 4px; }
        .date { font-size: 13px; color: #94a3b8; margin-top: 6px; }
        .player-banner { text-align: center; margin: 15px 0 22px 0; padding: 14px; background: #f5f3ff; border-radius: 10px; border: 2px solid #8b5cf6; }
        .player-label { font-size: 13px; color: #6d28d9; text-transform: uppercase; font-weight: 700; display: block; margin-bottom: 4px; }
        .player-name { font-size: 26px; color: #4c1d95; text-transform: uppercase; letter-spacing: 1px; }
        .card { border: 1px solid #cbd5e1; border-radius: 12px; padding: 18px; margin-bottom: 20px; background: #f8fafc; }
        .grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px; margin-bottom: 20px; }
        .stat-box { border: 1px solid #e2e8f0; border-radius: 10px; padding: 16px; text-align: center; background: #ffffff; box-shadow: 0 1px 3px rgba(0,0,0,0.05); }
        .stat-val { font-size: 28px; font-weight: 800; color: #4c1d95; margin-bottom: 4px; }
        .stat-label { font-size: 12px; color: #64748b; text-transform: uppercase; font-weight: 700; }
        .row-item { display: flex; justify-content: space-between; padding: 10px 0; border-bottom: 1px dashed #e2e8f0; font-size: 15px; }
        .row-item:last-child { border-bottom: none; }
        .status-ok { color: #059669; font-weight: 700; }
        .footer { text-align: center; margin-top: 36px; font-size: 12px; color: #94a3b8; border-top: 1px solid #e2e8f0; padding-top: 16px; line-height: 1.5; }
        .print-btn { display: inline-block; padding: 10px 20px; background: #7c3aed; color: #ffffff; border: none; border-radius: 20px; font-weight: bold; cursor: pointer; margin-bottom: 20px; }
        @media print { .print-btn { display: none; } }
    </style>
</head>
<body>
    <div style="text-align: right;">
        <button class="print-btn" onclick="window.print()">🖨️ Imprimir / Salvar PDF</button>
    </div>
    <div class="header">
        <div class="logo">🎙️ Fala Eh!</div>
        <div class="tagline">Certificado de Desempenho e Relatório de Missão</div>
        <div class="date">Emitido em: ${dataStr}</div>
    </div>

    <div class="player-banner">
        <span class="player-label">Certificado de Conclusão de Missão concedido a:</span>
        <strong class="player-name">⭐ ${escapeHtml(playerName)} ⭐</strong>
    </div>

    <div class="card">
        <div class="row-item"><strong>Mundo / Nível:</strong> <span>${currentWorld.icon} ${currentWorld.name} (${currentWorld.badge})</span></div>
        <div class="row-item"><strong>Fase Concluída:</strong> <span class="status-ok">Sim 🏆 (100%)</span></div>
        <div class="row-item"><strong>Próximo Nível Desbloqueado:</strong> <span style="color: #2563eb; font-weight: 700;">${nextLevelName}</span></div>
    </div>

    <div class="grid">
        <div class="stat-box">
            <div class="stat-val">⭐ ${totalXp} XP</div>
            <div class="stat-label">Pontuação Total Conquistada</div>
        </div>
        <div class="stat-box">
            <div class="stat-val">🎯 ${accuracy}%</div>
            <div class="stat-label">Taxa de Acertos (Acurácia)</div>
        </div>
        <div class="stat-box">
            <div class="stat-val">🔥 ${bestStreak}</div>
            <div class="stat-label">Maior Sequência de Acertos</div>
        </div>
        <div class="stat-box">
            <div class="stat-val">📝 ${exercisesCompleted}</div>
            <div class="stat-label">Exercícios Realizados</div>
        </div>
    </div>

    <div class="card">
        <div class="row-item"><strong>Total de Acertos:</strong> <span>${hits} acertos</span></div>
        <div class="row-item"><strong>Total de Tentativas:</strong> <span>${attempts} tentativas</span></div>
    </div>

    <div class="footer">
        🌱 <strong>Fala Eh</strong> — Jogo Educativo de Exercícios Fonoaudiológicos<br/>
        Ambiente gamificado para desenvolvimento e treino lúdico da fala.
    </div>

    <script>
        setTimeout(function() {
            window.print();
        }, 300);
    </script>
</body>
</html>
            `);
            printWin.document.close();
        } else {
            window.print();
        }
    } catch (err) {
        console.error("Falha ao abrir visualização de PDF:", err);
        window.print();
    }
}

// Geração local de imagem PNG em alta resolução utilizando HTML5 Canvas
function exportReportImage() {
    try {
        showToast("🖼️", "Gerando imagem do relatório...");
        const currentWorld = WORLDS[state.currentLevel] || { name: "Iniciante", badge: "NÍVEL 1", icon: "🌱" };
        const report = state.report;
        const totalXp = report?.totalXp ?? state.xp ?? 0;
        const accuracy = report?.accuracy ?? 100;
        const bestStreak = report?.bestStreak ?? state.streak ?? 0;
        const exercisesCompleted = report?.exercisesTotal ?? state.totalExercises ?? 0;
        const hits = report?.hits ?? state.totalExercises ?? 0;
        const attempts = report?.attempts ?? state.totalExercises ?? 0;
        const nextLevelName = state.completion?.nextLevel && WORLDS[state.completion.nextLevel]
            ? WORLDS[state.completion.nextLevel].name
            : "Missão Final Concluída! 🎖️";

        const playerName = getReportPlayerName();

        const canvas = document.createElement("canvas");
        canvas.width = 1200;
        canvas.height = 950;
        const ctx = canvas.getContext("2d");
        if (!ctx) {
            throw new Error("Canvas 2D context não suportado");
        }

        // 1. Fundo Espacial com Gradiente
        const bgGrad = ctx.createRadialGradient(600, 200, 50, 600, 475, 750);
        bgGrad.addColorStop(0, "#311042");
        bgGrad.addColorStop(0.5, "#1e1b4b");
        bgGrad.addColorStop(1, "#0f172a");
        ctx.fillStyle = bgGrad;
        ctx.fillRect(0, 0, 1200, 950);

        // 2. Borda Estilizada
        ctx.strokeStyle = "rgba(139, 92, 246, 0.4)";
        ctx.lineWidth = 8;
        ctx.strokeRect(30, 30, 1140, 890);

        ctx.strokeStyle = "rgba(245, 158, 11, 0.6)";
        ctx.lineWidth = 2;
        ctx.strokeRect(40, 40, 1120, 870);

        // 3. Estrelas decorativas
        ctx.fillStyle = "#fef08a";
        const stars = [
            [100, 100, 3], [1100, 120, 4], [150, 850, 2], [1050, 830, 3],
            [250, 200, 2], [950, 220, 3], [120, 480, 2], [1080, 510, 2]
        ];
        stars.forEach(([x, y, r]) => {
            ctx.beginPath();
            ctx.arc(x, y, r, 0, Math.PI * 2);
            ctx.fill();
        });

        // 4. Cabeçalho
        ctx.textAlign = "center";
        ctx.fillStyle = "#a78bfa";
        ctx.font = "bold 20px system-ui, -apple-system, sans-serif";
        ctx.fillText("✨ FALA EH • RELATÓRIO DA MISSÃO", 600, 75);

        ctx.fillStyle = "#f59e0b";
        ctx.font = "900 38px system-ui, -apple-system, sans-serif";
        ctx.fillText("CERTIFICADO DE CONCLUSÃO", 600, 118);

        // Banner Destaque do Nome Personalizado
        const nameBannerW = 860;
        const nameBannerH = 70;
        const nameBannerX = 600 - nameBannerW / 2;
        const nameBannerY = 140;

        ctx.fillStyle = "rgba(76, 29, 149, 0.85)";
        ctx.strokeStyle = "#fbbf24";
        ctx.lineWidth = 3;
        drawRoundRect(ctx, nameBannerX, nameBannerY, nameBannerW, nameBannerH, 16);
        ctx.fill();
        ctx.stroke();

        ctx.fillStyle = "#e2e8f0";
        ctx.font = "bold 13px system-ui, -apple-system, sans-serif";
        ctx.fillText("CONFERIDO COM HONRAS A:", 600, nameBannerY + 22);

        ctx.fillStyle = "#fef08a";
        const nameLine = `⭐ ${playerName.toUpperCase()} ⭐`;
        let nameFontSize = 28;
        ctx.font = `900 ${nameFontSize}px system-ui, -apple-system, sans-serif`;
        while (nameFontSize > 14 && ctx.measureText(nameLine).width > nameBannerW - 60) {
            nameFontSize -= 1;
            ctx.font = `900 ${nameFontSize}px system-ui, -apple-system, sans-serif`;
        }
        ctx.fillText(nameLine, 600, nameBannerY + 54);

        ctx.fillStyle = "#e2e8f0";
        ctx.font = "500 20px system-ui, -apple-system, sans-serif";
        ctx.fillText(`Mundo: ${currentWorld.name} (${currentWorld.badge})`, 600, 240);

        // 5. Grid de Cards de Estatísticas com drawRoundRect seguro
        const drawCard = (x, y, w, h, icon, label, val, color) => {
            ctx.fillStyle = "rgba(30, 27, 75, 0.75)";
            ctx.strokeStyle = "rgba(139, 92, 246, 0.35)";
            ctx.lineWidth = 3;
            drawRoundRect(ctx, x, y, w, h, 20);
            ctx.fill();
            ctx.stroke();

            ctx.textAlign = "center";
            ctx.font = "34px system-ui, -apple-system, sans-serif";
            ctx.fillText(icon, x + w / 2, y + 55);

            ctx.fillStyle = color;
            ctx.font = "bold 38px system-ui, -apple-system, sans-serif";
            ctx.fillText(val, x + w / 2, y + 110);

            ctx.fillStyle = "#94a3b8";
            ctx.font = "600 18px system-ui, -apple-system, sans-serif";
            ctx.fillText(label, x + w / 2, y + 145);
        };

        const cardW = 240;
        const cardH = 175;
        const startY = 260;
        const gap = 30;
        const startX = 600 - (cardW * 4 + gap * 3) / 2;

        drawCard(startX, startY, cardW, cardH, "⭐", "XP CONQUISTADO", `${totalXp} XP`, "#fbbf24");
        drawCard(startX + cardW + gap, startY, cardW, cardH, "🎯", "ACURÁCIA", `${accuracy}%`, "#34d399");
        drawCard(startX + (cardW + gap) * 2, startY, cardW, cardH, "🔥", "MAIOR SEQUÊNCIA", `${bestStreak}`, "#f87171");
        drawCard(startX + (cardW + gap) * 3, startY, cardW, cardH, "📝", "EXERCÍCIOS", `${exercisesCompleted}`, "#38bdf8");

        // 6. Painel de Detalhes Adicionais com drawRoundRect seguro
        const panelY = 480;
        const panelW = 1050;
        const panelH = 260;
        const panelX = 600 - panelW / 2;

        ctx.fillStyle = "rgba(15, 23, 42, 0.8)";
        ctx.strokeStyle = "rgba(167, 139, 250, 0.25)";
        ctx.lineWidth = 2;
        drawRoundRect(ctx, panelX, panelY, panelW, panelH, 24);
        ctx.fill();
        ctx.stroke();

        ctx.textAlign = "left";
        ctx.fillStyle = "#e2e8f0";
        ctx.font = "bold 26px system-ui, -apple-system, sans-serif";
        ctx.fillText("🏆 Fase Concluída:", panelX + 50, panelY + 65);

        ctx.fillStyle = "#34d399";
        ctx.fillText("SIM (100% dos desafios superados)", panelX + 310, panelY + 65);

        ctx.fillStyle = "#e2e8f0";
        ctx.fillText("🚀 Próximo Nível:", panelX + 50, panelY + 125);

        ctx.fillStyle = "#60a5fa";
        ctx.fillText(`${nextLevelName}`, panelX + 280, panelY + 125);

        ctx.fillStyle = "#94a3b8";
        ctx.font = "500 22px system-ui, -apple-system, sans-serif";
        ctx.fillText(`✅ Total de Acertos: ${hits} acertos`, panelX + 50, panelY + 185);
        ctx.fillText(`🔄 Total de Tentativas: ${attempts} tentativas`, panelX + 450, panelY + 185);

        const dataStr = new Date().toLocaleDateString("pt-BR", { day: "2-digit", month: "2-digit", year: "numeric" });
        ctx.fillText(`📅 Data: ${dataStr}`, panelX + 830, panelY + 185);

        // 7. Rodapé
        ctx.textAlign = "center";
        ctx.fillStyle = "#64748b";
        ctx.font = "16px system-ui, -apple-system, sans-serif";
        ctx.fillText("🌱 Fala Eh — Jogo Educativo de Exercícios Fonoaudiológicos • Treino Lúdico da Fala", 600, 850);

        // 8. Download Automático sem envio a servidores
        const cleanName = playerName
            .toLowerCase()
            .normalize("NFD")
            .replace(/[\u0300-\u036f]/g, "")
            .replace(/[^a-z0-9]/g, "-")
            .replace(/-+/g, "-")
            .replace(/^-|-$/g, "") || "resultado";

        const dataUrl = canvas.toDataURL("image/png");
        const a = document.createElement("a");
        a.href = dataUrl;
        a.download = `falaeh-certificado-${cleanName}.png`;
        document.body.appendChild(a);
        a.click();
        setTimeout(() => {
            document.body.removeChild(a);
        }, 100);

        showToast("🖼️", "Imagem personalizada gerada e baixada com sucesso!");
    } catch (err) {
        console.error("Falha ao gerar imagem do relatório:", err);
        showToast("⚠️", "Não foi possível gerar a imagem no navegador.");
    }
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

    if (btnExportPdf) {
        btnExportPdf.addEventListener("click", exportReportPDF);
    }

    if (btnExportImage) {
        btnExportImage.addEventListener("click", exportReportImage);
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

