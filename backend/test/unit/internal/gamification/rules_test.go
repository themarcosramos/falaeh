package gamification_test

import (
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/gamification"
)

func TestDefaultRules(t *testing.T) {
	rules := gamification.DefaultRules()

	if rules.BaseCorrectXP != 100 {
		t.Errorf("BaseCorrectXP esperado 100, obteve %d", rules.BaseCorrectXP)
	}
	if rules.FirstAttemptBonusXP != 50 {
		t.Errorf("FirstAttemptBonusXP esperado 50, obteve %d", rules.FirstAttemptBonusXP)
	}
	if rules.Streak3BonusXP != 50 {
		t.Errorf("Streak3BonusXP esperado 50, obteve %d", rules.Streak3BonusXP)
	}
	if rules.Streak5BonusXP != 100 {
		t.Errorf("Streak5BonusXP esperado 100, obteve %d", rules.Streak5BonusXP)
	}
	if rules.PhaseCompletionBonusXP != 300 {
		t.Errorf("PhaseCompletionBonusXP esperado 300, obteve %d", rules.PhaseCompletionBonusXP)
	}
	if rules.LevelCompletionBonusXP != 500 {
		t.Errorf("LevelCompletionBonusXP esperado 500, obteve %d", rules.LevelCompletionBonusXP)
	}
	if rules.Streak3Threshold != 3 || rules.Streak5Threshold != 5 {
		t.Errorf("limiares de streak inesperados: 3=%d, 5=%d", rules.Streak3Threshold, rules.Streak5Threshold)
	}
}

func TestNextLevel(t *testing.T) {
	tests := []struct {
		name         string
		current      exercise.Level
		expectedNext exercise.Level
		hasNext      bool
	}{
		{
			name:         "beginner avança para intermediate",
			current:      exercise.LevelBeginner,
			expectedNext: exercise.LevelIntermediate,
			hasNext:      true,
		},
		{
			name:         "intermediate avança para advanced",
			current:      exercise.LevelIntermediate,
			expectedNext: exercise.LevelAdvanced,
			hasNext:      true,
		},
		{
			name:         "advanced não possui próximo nível",
			current:      exercise.LevelAdvanced,
			expectedNext: "",
			hasNext:      false,
		},
		{
			name:         "nível inválido não possui próximo",
			current:      exercise.Level("invalido"),
			expectedNext: "",
			hasNext:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next, hasNext := gamification.NextLevel(tt.current)
			if hasNext != tt.hasNext {
				t.Fatalf("hasNext = %v, esperado %v", hasNext, tt.hasNext)
			}
			if next != tt.expectedNext {
				t.Fatalf("next = %s, esperado %s", next, tt.expectedNext)
			}
		})
	}
}
