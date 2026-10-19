package gamification

import "github.com/themarcosramos/falaeh/backend/internal/exercise"

// Quantidades do banco de exercícios de cada fase do jogo.
const (
	BeginnerTotalExercises     = 40
	IntermediateTotalExercises = 50
	AdvancedTotalExercises     = 60
	FinalTotalExercises        = 10
)

// Acertos em exercícios diferentes necessários para concluir cada fase.
const (
	BeginnerRequiredHits     = 25
	IntermediateRequiredHits = 35
	AdvancedRequiredHits     = 40
	FinalRequiredHits        = 10
)

// RequiredHits informa quantos acertos em exercícios diferentes são
// necessários para concluir a fase informada.
func RequiredHits(level exercise.Level) int {
	switch level {
	case exercise.LevelBeginner:
		return BeginnerRequiredHits
	case exercise.LevelIntermediate:
		return IntermediateRequiredHits
	case exercise.LevelAdvanced:
		return AdvancedRequiredHits
	case exercise.LevelFinal:
		return FinalRequiredHits
	default:
		return 0
	}
}
