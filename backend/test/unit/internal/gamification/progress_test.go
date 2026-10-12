package gamification_test

import (
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/gamification"
)

func TestProgress_InitialState(t *testing.T) {
	p := gamification.NewProgress()

	if p.TotalXP != 0 {
		t.Errorf("TotalXP inicial esperado 0, obteve %d", p.TotalXP)
	}
	if p.CurrentLevel != exercise.LevelBeginner {
		t.Errorf("CurrentLevel inicial esperado beginner, obteve %s", p.CurrentLevel)
	}
	if !p.IsLevelUnlocked(exercise.LevelBeginner) {
		t.Errorf("nível beginner deveria estar desbloqueado inicialmente")
	}
	if p.IsLevelUnlocked(exercise.LevelIntermediate) {
		t.Errorf("nível intermediate NÃO deveria estar desbloqueado inicialmente")
	}
	if p.IsLevelUnlocked(exercise.LevelAdvanced) {
		t.Errorf("nível advanced NÃO deveria estar desbloqueado inicialmente")
	}
	if len(p.CompletedLevels) != 0 {
		t.Errorf("CompletedLevels deveria estar vazio inicialmente")
	}
	if len(p.Achievements) != 0 {
		t.Errorf("Achievements deveria estar vazio inicialmente")
	}
}

func TestProgress_UnlockLevel(t *testing.T) {
	p := gamification.NewProgress()

	t.Run("desbloquear nível intermediário com sucesso", func(t *testing.T) {
		unlocked := p.UnlockLevel(exercise.LevelIntermediate)
		if !unlocked {
			t.Fatalf("esperava retorno true ao desbloquear intermediate")
		}
		if !p.IsLevelUnlocked(exercise.LevelIntermediate) {
			t.Errorf("intermediate deveria estar desbloqueado")
		}
	})

	t.Run("desbloquear nível já desbloqueado retorna false", func(t *testing.T) {
		unlocked := p.UnlockLevel(exercise.LevelIntermediate)
		if unlocked {
			t.Errorf("esperava retorno false ao tentar desbloquear nível já desbloqueado")
		}
	})

	t.Run("desbloquear nível inválido retorna false", func(t *testing.T) {
		unlocked := p.UnlockLevel(exercise.Level("invalido"))
		if unlocked {
			t.Errorf("esperava retorno false ao tentar desbloquear nível inválido")
		}
	})
}

func TestProgress_IsLevelCompleted(t *testing.T) {
	p := gamification.NewProgress()

	if p.IsLevelCompleted(exercise.LevelBeginner) {
		t.Errorf("nível beginner não deveria estar completo")
	}

	p.CompletedLevels = append(p.CompletedLevels, exercise.LevelBeginner)
	if !p.IsLevelCompleted(exercise.LevelBeginner) {
		t.Errorf("nível beginner deveria estar completo")
	}
}

func TestProgress_HasAchievement(t *testing.T) {
	p := gamification.NewProgress()

	if p.HasAchievement(gamification.AchievementFirstCorrect) {
		t.Errorf("não deveria possuir conquista inicialmente")
	}

	p.Achievements = append(p.Achievements, gamification.Achievement{
		ID:    gamification.AchievementFirstCorrect,
		Title: "Primeiro Passo",
	})

	if !p.HasAchievement(gamification.AchievementFirstCorrect) {
		t.Errorf("deveria possuir conquista")
	}
}
