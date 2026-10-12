package gamification_test

import (
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/gamification"
)

func TestCalculateAnswerXP(t *testing.T) {
	rules := gamification.DefaultRules()

	tests := []struct {
		name                string
		isCorrect           bool
		isFirstAttempt      bool
		currentStreak       int
		expectedBaseXP      int
		expectedFirstBonus  int
		expectedStreakBonus int
		expectedTotal       int
	}{
		{
			name:                "resposta incorreta não pontua",
			isCorrect:           false,
			isFirstAttempt:      true,
			currentStreak:       0,
			expectedBaseXP:      0,
			expectedFirstBonus:  0,
			expectedStreakBonus: 0,
			expectedTotal:       0,
		},
		{
			name:                "primeiro acerto na primeira tentativa sem streak bônus",
			isCorrect:           true,
			isFirstAttempt:      true,
			currentStreak:       1,
			expectedBaseXP:      100,
			expectedFirstBonus:  50,
			expectedStreakBonus: 0,
			expectedTotal:       150,
		},
		{
			name:                "acerto na segunda tentativa (sem bônus de primeira tentativa)",
			isCorrect:           true,
			isFirstAttempt:      false,
			currentStreak:       1,
			expectedBaseXP:      100,
			expectedFirstBonus:  0,
			expectedStreakBonus: 0,
			expectedTotal:       100,
		},
		{
			name:                "acerto atingindo streak 3 na primeira tentativa",
			isCorrect:           true,
			isFirstAttempt:      true,
			currentStreak:       3,
			expectedBaseXP:      100,
			expectedFirstBonus:  50,
			expectedStreakBonus: 50,
			expectedTotal:       200,
		},
		{
			name:                "acerto atingindo streak 3 na segunda tentativa",
			isCorrect:           true,
			isFirstAttempt:      false,
			currentStreak:       3,
			expectedBaseXP:      100,
			expectedFirstBonus:  0,
			expectedStreakBonus: 50,
			expectedTotal:       150,
		},
		{
			name:                "acerto com streak 4 (entre os limiares)",
			isCorrect:           true,
			isFirstAttempt:      true,
			currentStreak:       4,
			expectedBaseXP:      100,
			expectedFirstBonus:  50,
			expectedStreakBonus: 0,
			expectedTotal:       150,
		},
		{
			name:                "acerto atingindo streak 5 na primeira tentativa",
			isCorrect:           true,
			isFirstAttempt:      true,
			currentStreak:       5,
			expectedBaseXP:      100,
			expectedFirstBonus:  50,
			expectedStreakBonus: 100,
			expectedTotal:       250,
		},
		{
			name:                "acerto atingindo streak 5 na segunda tentativa",
			isCorrect:           true,
			isFirstAttempt:      false,
			currentStreak:       5,
			expectedBaseXP:      100,
			expectedFirstBonus:  0,
			expectedStreakBonus: 100,
			expectedTotal:       200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			breakdown := gamification.CalculateAnswerXP(rules, tt.isCorrect, tt.isFirstAttempt, tt.currentStreak)

			if breakdown.BaseXP != tt.expectedBaseXP {
				t.Errorf("BaseXP = %d, esperado %d", breakdown.BaseXP, tt.expectedBaseXP)
			}
			if breakdown.FirstAttemptBonus != tt.expectedFirstBonus {
				t.Errorf("FirstAttemptBonus = %d, esperado %d", breakdown.FirstAttemptBonus, tt.expectedFirstBonus)
			}
			if breakdown.StreakBonus != tt.expectedStreakBonus {
				t.Errorf("StreakBonus = %d, esperado %d", breakdown.StreakBonus, tt.expectedStreakBonus)
			}
			if breakdown.Total != tt.expectedTotal {
				t.Errorf("Total = %d, esperado %d", breakdown.Total, tt.expectedTotal)
			}
		})
	}
}

func TestCalculatePhaseCompletionXP(t *testing.T) {
	rules := gamification.DefaultRules()
	breakdown := gamification.CalculatePhaseCompletionXP(rules)

	if breakdown.PhaseBonus != 300 || breakdown.Total != 300 {
		t.Errorf("bônus de fase inesperado: %+v", breakdown)
	}
}

func TestCalculateLevelCompletionXP(t *testing.T) {
	rules := gamification.DefaultRules()
	breakdown := gamification.CalculateLevelCompletionXP(rules)

	if breakdown.LevelBonus != 500 || breakdown.Total != 500 {
		t.Errorf("bônus de nível inesperado: %+v", breakdown)
	}
}
