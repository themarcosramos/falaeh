package gamification_test

import (
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/gamification"
)

// TestRequiredHits_Progressao valida as regras centralizadas de progressão por mundo:
// Iniciante 25/40, Intermediário 35/50, Avançado 40/60 e Desafio Final 10/10.
func TestRequiredHits_Progressao(t *testing.T) {
	tests := []struct {
		name     string
		level    exercise.Level
		required int
	}{
		{"iniciante exige 25 acertos", exercise.LevelBeginner, 25},
		{"intermediário exige 35 acertos", exercise.LevelIntermediate, 35},
		{"avançado exige 40 acertos", exercise.LevelAdvanced, 40},
		{"desafio final exige 10 acertos", exercise.LevelFinal, 10},
		{"nível inválido não exige acertos", exercise.Level("invalido"), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gamification.RequiredHits(tt.level); got != tt.required {
				t.Errorf("RequiredHits(%s) = %d, esperado %d", tt.level, got, tt.required)
			}
		})
	}

	if gamification.BeginnerTotalExercises != 40 || gamification.IntermediateTotalExercises != 50 ||
		gamification.AdvancedTotalExercises != 60 || gamification.FinalTotalExercises != 10 {
		t.Errorf("tamanhos de banco inesperados: %d/%d/%d/%d",
			gamification.BeginnerTotalExercises, gamification.IntermediateTotalExercises,
			gamification.AdvancedTotalExercises, gamification.FinalTotalExercises)
	}
}
