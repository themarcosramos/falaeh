package gamification

import "github.com/themarcosramos/falaeh/backend/internal/exercise"

// Progress representa o estado de progresso mantido em memória durante a sessão.
type Progress struct {
	TotalXP          int              `json:"totalXp"`
	CurrentStreak    int              `json:"currentStreak"`
	BestStreak       int              `json:"bestStreak"`
	TotalHits        int              `json:"totalHits"`
	TotalMisses      int              `json:"totalMisses"`
	TotalAttempts    int              `json:"totalAttempts"`
	CurrentLevel     exercise.Level   `json:"currentLevel"`
	UnlockedLevels   []exercise.Level `json:"unlockedLevels"`
	CompletedLevels  []exercise.Level `json:"completedLevels"`
	Achievements     []Achievement    `json:"achievements"`
	exerciseAttempts map[string]int
}

// NewProgress inicializa um novo progresso de jogador com o nível iniciante desbloqueado.
func NewProgress() *Progress {
	return &Progress{
		CurrentLevel:     exercise.LevelBeginner,
		UnlockedLevels:   []exercise.Level{exercise.LevelBeginner},
		CompletedLevels:  make([]exercise.Level, 0),
		Achievements:     make([]Achievement, 0),
		exerciseAttempts: make(map[string]int),
	}
}

// IsLevelUnlocked verifica se um nível já está disponível para o jogador.
func (p *Progress) IsLevelUnlocked(level exercise.Level) bool {
	for _, l := range p.UnlockedLevels {
		if l == level {
			return true
		}
	}
	return false
}

// UnlockLevel desbloqueia um novo nível caso seja válido e ainda não esteja desbloqueado.
func (p *Progress) UnlockLevel(level exercise.Level) bool {
	if !level.IsValid() || p.IsLevelUnlocked(level) {
		return false
	}
	p.UnlockedLevels = append(p.UnlockedLevels, level)
	return true
}

// IsLevelCompleted verifica se um nível já foi finalizado.
func (p *Progress) IsLevelCompleted(level exercise.Level) bool {
	for _, l := range p.CompletedLevels {
		if l == level {
			return true
		}
	}
	return false
}

// HasAchievement verifica se o jogador já obteve uma determinada conquista.
func (p *Progress) HasAchievement(id AchievementID) bool {
	for _, a := range p.Achievements {
		if a.ID == id {
			return true
		}
	}
	return false
}
