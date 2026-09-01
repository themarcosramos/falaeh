package gamification

import "github.com/themarcosramos/falaeh/backend/internal/exercise"

// Constantes padrão das regras de pontuação de gamificação.
const (
	DefaultBaseCorrectXP          = 100
	DefaultFirstAttemptBonusXP    = 50
	DefaultStreak3BonusXP         = 50
	DefaultStreak5BonusXP         = 100
	DefaultPhaseCompletionBonusXP = 300
	DefaultLevelCompletionBonusXP = 500

	DefaultStreak3Threshold = 3
	DefaultStreak5Threshold = 5
)

// Rules centraliza os valores e multiplicadores de gamificação da aplicação.
type Rules struct {
	BaseCorrectXP          int `json:"baseCorrectXp"`
	FirstAttemptBonusXP    int `json:"firstAttemptBonusXp"`
	Streak3BonusXP         int `json:"streak3BonusXp"`
	Streak5BonusXP         int `json:"streak5BonusXp"`
	PhaseCompletionBonusXP int `json:"phaseCompletionBonusXp"`
	LevelCompletionBonusXP int `json:"levelCompletionBonusXp"`
	Streak3Threshold       int `json:"streak3Threshold"`
	Streak5Threshold       int `json:"streak5Threshold"`
}

// DefaultRules retorna as regras de pontuação padrão do jogo.
func DefaultRules() Rules {
	return Rules{
		BaseCorrectXP:          DefaultBaseCorrectXP,
		FirstAttemptBonusXP:    DefaultFirstAttemptBonusXP,
		Streak3BonusXP:         DefaultStreak3BonusXP,
		Streak5BonusXP:         DefaultStreak5BonusXP,
		PhaseCompletionBonusXP: DefaultPhaseCompletionBonusXP,
		LevelCompletionBonusXP: DefaultLevelCompletionBonusXP,
		Streak3Threshold:       DefaultStreak3Threshold,
		Streak5Threshold:       DefaultStreak5Threshold,
	}
}

// NextLevel determina o próximo nível de acordo com a progressão canônica do jogo.
func NextLevel(current exercise.Level) (exercise.Level, bool) {
	switch current {
	case exercise.LevelBeginner:
		return exercise.LevelIntermediate, true
	case exercise.LevelIntermediate:
		return exercise.LevelAdvanced, true
	case exercise.LevelAdvanced:
		return "", false
	default:
		return "", false
	}
}
