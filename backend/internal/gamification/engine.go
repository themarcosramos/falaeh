package gamification

import (
	"sync"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
)

// AttemptResult resume o impacto de uma resposta no progresso e XP do jogador.
type AttemptResult struct {
	ExerciseID      string        `json:"exerciseId"`
	Correct         bool          `json:"correct"`
	EarnedXP        int           `json:"earnedXp"`
	Breakdown       XPBreakdown   `json:"breakdown"`
	TotalXP         int           `json:"totalXp"`
	Streak          int           `json:"streak"`
	NewAchievements []Achievement `json:"newAchievements,omitempty"`
}

// LevelCompletionResult resume as recompensas ao concluir um nível/mundo.
type LevelCompletionResult struct {
	Level           exercise.Level `json:"level"`
	EarnedXP        int            `json:"earnedXp"`
	Breakdown       XPBreakdown    `json:"breakdown"`
	TotalXP         int            `json:"totalXp"`
	NextLevel       exercise.Level `json:"nextLevel,omitempty"`
	NextUnlocked    bool           `json:"nextUnlocked"`
	NewAchievements []Achievement  `json:"newAchievements,omitempty"`
}

// Engine gerencia as regras de negócio de pontuação, progressão e conquistas durante a sessão.
type Engine struct {
	mu       sync.RWMutex
	rules    Rules
	progress *Progress
}

// NewEngine inicializa a engine com regras e estado padrão.
func NewEngine(rules Rules) *Engine {
	return &Engine{
		rules:    rules,
		progress: NewProgress(),
	}
}

// GetProgress retorna uma cópia segura do estado atual de progresso.
func (e *Engine) GetProgress() Progress {
	e.mu.RLock()
	defer e.mu.RUnlock()

	unlocked := make([]exercise.Level, len(e.progress.UnlockedLevels))
	copy(unlocked, e.progress.UnlockedLevels)

	completed := make([]exercise.Level, len(e.progress.CompletedLevels))
	copy(completed, e.progress.CompletedLevels)

	achievements := make([]Achievement, len(e.progress.Achievements))
	copy(achievements, e.progress.Achievements)

	return Progress{
		TotalXP:         e.progress.TotalXP,
		CurrentStreak:   e.progress.CurrentStreak,
		BestStreak:      e.progress.BestStreak,
		TotalHits:       e.progress.TotalHits,
		TotalMisses:     e.progress.TotalMisses,
		TotalAttempts:   e.progress.TotalAttempts,
		CurrentLevel:    e.progress.CurrentLevel,
		UnlockedLevels:  unlocked,
		CompletedLevels: completed,
		Achievements:    achievements,
	}
}

// ProcessAnswer processa uma resposta do usuário e atualiza contadores, streak, XP e conquistas.
func (e *Engine) ProcessAnswer(exerciseID string, isCorrect bool) AttemptResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.progress.TotalAttempts++
	currentAttempts := e.progress.exerciseAttempts[exerciseID]
	isFirstAttempt := currentAttempts == 0
	e.progress.exerciseAttempts[exerciseID]++

	var newAchievements []Achievement

	if isCorrect {
		e.progress.TotalHits++
		e.progress.CurrentStreak++
		if e.progress.CurrentStreak > e.progress.BestStreak {
			e.progress.BestStreak = e.progress.CurrentStreak
		}

		// Conquistas de primeiro acerto e streak
		if !e.progress.HasAchievement(AchievementFirstCorrect) {
			ach := Achievement{
				ID:          AchievementFirstCorrect,
				Title:       "Primeiro Passo",
				Description: "Acertou seu primeiro exercício fonoaudiológico!",
				Icon:        "⭐",
			}
			e.progress.Achievements = append(e.progress.Achievements, ach)
			newAchievements = append(newAchievements, ach)
		}

		if e.progress.CurrentStreak >= e.rules.Streak3Threshold && !e.progress.HasAchievement(AchievementStreak3) {
			ach := Achievement{
				ID:          AchievementStreak3,
				Title:       "Na Sequência!",
				Description: "Alcançou uma sequência de 3 acertos seguidos!",
				Icon:        "🔥",
			}
			e.progress.Achievements = append(e.progress.Achievements, ach)
			newAchievements = append(newAchievements, ach)
		}

		if e.progress.CurrentStreak >= e.rules.Streak5Threshold && !e.progress.HasAchievement(AchievementStreak5) {
			ach := Achievement{
				ID:          AchievementStreak5,
				Title:       "Imparável!",
				Description: "Incrível! Alcançou 5 acertos seguidos sem errar!",
				Icon:        "🚀",
			}
			e.progress.Achievements = append(e.progress.Achievements, ach)
			newAchievements = append(newAchievements, ach)
		}
	} else {
		e.progress.TotalMisses++
		e.progress.CurrentStreak = 0
	}

	breakdown := CalculateAnswerXP(e.rules, isCorrect, isFirstAttempt, e.progress.CurrentStreak)
	e.progress.TotalXP += breakdown.Total

	return AttemptResult{
		ExerciseID:      exerciseID,
		Correct:         isCorrect,
		EarnedXP:        breakdown.Total,
		Breakdown:       breakdown,
		TotalXP:         e.progress.TotalXP,
		Streak:          e.progress.CurrentStreak,
		NewAchievements: newAchievements,
	}
}

// CompletePhase concede bônus de fase concluída.
func (e *Engine) CompletePhase(phaseID string) XPBreakdown {
	e.mu.Lock()
	defer e.mu.Unlock()

	breakdown := CalculatePhaseCompletionXP(e.rules)
	e.progress.TotalXP += breakdown.Total
	return breakdown
}

// CompleteLevel concede o bônus de conclusão de nível e desbloqueia o próximo nível.
func (e *Engine) CompleteLevel(level exercise.Level) LevelCompletionResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.progress.IsLevelCompleted(level) {
		e.progress.CompletedLevels = append(e.progress.CompletedLevels, level)
	}

	breakdown := CalculateLevelCompletionXP(e.rules)
	e.progress.TotalXP += breakdown.Total

	var newAchievements []Achievement

	// Conquistas de nível
	var levelAch AchievementID
	var title, desc string
	switch level {
	case exercise.LevelBeginner:
		levelAch = AchievementBeginnerComplete
		title = "Explorador Inicial"
		desc = "Concluiu todos os desafios do Nível Iniciante!"
	case exercise.LevelIntermediate:
		levelAch = AchievementIntermediateComplete
		title = "Aventureiro da Articulação"
		desc = "Concluiu todos os desafios do Nível Intermediário!"
	case exercise.LevelAdvanced:
		levelAch = AchievementAdvancedComplete
		title = "Mestre da Fala"
		desc = "Parabéns! Concluiu todos os desafios do Nível Avançado!"
	}

	if levelAch != "" && !e.progress.HasAchievement(levelAch) {
		ach := Achievement{
			ID:          levelAch,
			Title:       title,
			Description: desc,
			Icon:        "🏆",
		}
		e.progress.Achievements = append(e.progress.Achievements, ach)
		newAchievements = append(newAchievements, ach)
	}

	next, hasNext := NextLevel(level)
	nextUnlocked := false
	if hasNext {
		nextUnlocked = e.progress.UnlockLevel(next)
		if nextUnlocked {
			e.progress.CurrentLevel = next
		}
	}

	return LevelCompletionResult{
		Level:           level,
		EarnedXP:        breakdown.Total,
		Breakdown:       breakdown,
		TotalXP:         e.progress.TotalXP,
		NextLevel:       next,
		NextUnlocked:    nextUnlocked,
		NewAchievements: newAchievements,
	}
}

// Reset reinicializa a engine para o estado inicial.
func (e *Engine) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.progress = NewProgress()
}
