package gamification_test

import (
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/gamification"
)

func TestEngine_ProcessAnswer_Flow(t *testing.T) {
	rules := gamification.DefaultRules()
	engine := gamification.NewEngine(rules)

	t.Run("1º acerto na 1ª tentativa concede 150 XP e conquista de primeiro acerto", func(t *testing.T) {
		res := engine.ProcessAnswer("beg-001", true)
		if !res.Correct {
			t.Errorf("esperava resultado correto")
		}
		if res.EarnedXP != 150 {
			t.Errorf("EarnedXP = %d, esperado 150 (100 base + 50 primeira tentativa)", res.EarnedXP)
		}
		if res.TotalXP != 150 {
			t.Errorf("TotalXP = %d, esperado 150", res.TotalXP)
		}
		if res.Streak != 1 {
			t.Errorf("Streak = %d, esperado 1", res.Streak)
		}
		if len(res.NewAchievements) != 1 || res.NewAchievements[0].ID != gamification.AchievementFirstCorrect {
			t.Errorf("esperava conquista FirstCorrect, obteve %+v", res.NewAchievements)
		}
	})

	t.Run("2º acerto consecutivo na 1ª tentativa de outro exercício concede 150 XP e streak 2", func(t *testing.T) {
		res := engine.ProcessAnswer("beg-002", true)
		if res.EarnedXP != 150 {
			t.Errorf("EarnedXP = %d, esperado 150", res.EarnedXP)
		}
		if res.TotalXP != 300 {
			t.Errorf("TotalXP = %d, esperado 300", res.TotalXP)
		}
		if res.Streak != 2 {
			t.Errorf("Streak = %d, esperado 2", res.Streak)
		}
	})

	t.Run("3º acerto consecutivo atinge streak 3 e concede 200 XP e conquista Streak3", func(t *testing.T) {
		res := engine.ProcessAnswer("beg-003", true)
		if res.EarnedXP != 200 {
			t.Errorf("EarnedXP = %d, esperado 200 (100 base + 50 primeira + 50 streak 3)", res.EarnedXP)
		}
		if res.TotalXP != 500 {
			t.Errorf("TotalXP = %d, esperado 500", res.TotalXP)
		}
		if res.Streak != 3 {
			t.Errorf("Streak = %d, esperado 3", res.Streak)
		}
		if len(res.NewAchievements) != 1 || res.NewAchievements[0].ID != gamification.AchievementStreak3 {
			t.Errorf("esperava conquista Streak3, obteve %+v", res.NewAchievements)
		}
	})

	t.Run("erro reseta o streak para 0 e não pontua", func(t *testing.T) {
		res := engine.ProcessAnswer("beg-004", false)
		if res.Correct {
			t.Errorf("esperava resposta incorreta")
		}
		if res.EarnedXP != 0 {
			t.Errorf("EarnedXP = %d, esperado 0", res.EarnedXP)
		}
		if res.TotalXP != 500 {
			t.Errorf("TotalXP = %d, esperado 500", res.TotalXP)
		}
		if res.Streak != 0 {
			t.Errorf("Streak = %d, esperado 0", res.Streak)
		}
	})

	t.Run("segunda tentativa no mesmo exercício (beg-004) com acerto concede apenas 100 XP", func(t *testing.T) {
		res := engine.ProcessAnswer("beg-004", true)
		if res.EarnedXP != 100 {
			t.Errorf("EarnedXP = %d, esperado 100 (apenas base, sem bônus de 1ª tentativa)", res.EarnedXP)
		}
		if res.TotalXP != 600 {
			t.Errorf("TotalXP = %d, esperado 600", res.TotalXP)
		}
		if res.Streak != 1 {
			t.Errorf("Streak = %d, esperado 1", res.Streak)
		}
	})
}

func TestEngine_Streak5_Achievement(t *testing.T) {
	rules := gamification.DefaultRules()
	engine := gamification.NewEngine(rules)

	// Realiza 4 acertos
	for i := 1; i <= 4; i++ {
		engine.ProcessAnswer(string(rune('a'+i)), true)
	}

	// 5º acerto
	res := engine.ProcessAnswer("ex-5", true)
	if res.Streak != 5 {
		t.Fatalf("Streak = %d, esperado 5", res.Streak)
	}
	if res.EarnedXP != 250 {
		t.Errorf("EarnedXP = %d, esperado 250 (100 base + 50 primeira + 100 streak 5)", res.EarnedXP)
	}

	foundStreak5 := false
	for _, ach := range res.NewAchievements {
		if ach.ID == gamification.AchievementStreak5 {
			foundStreak5 = true
			break
		}
	}
	if !foundStreak5 {
		t.Errorf("esperava conquista AchievementStreak5")
	}
}

func TestEngine_CompletePhase(t *testing.T) {
	rules := gamification.DefaultRules()
	engine := gamification.NewEngine(rules)

	breakdown := engine.CompletePhase("phase-1")
	if breakdown.Total != 300 || breakdown.PhaseBonus != 300 {
		t.Errorf("esperava 300 XP de bônus de fase, obteve %+v", breakdown)
	}

	progress := engine.GetProgress()
	if progress.TotalXP != 300 {
		t.Errorf("TotalXP = %d, esperado 300", progress.TotalXP)
	}
}

func TestEngine_CompleteLevel_Progression(t *testing.T) {
	rules := gamification.DefaultRules()
	engine := gamification.NewEngine(rules)

	t.Run("conclusão do nível beginner desbloqueia intermediate", func(t *testing.T) {
		res := engine.CompleteLevel(exercise.LevelBeginner)
		if res.EarnedXP != 500 {
			t.Errorf("EarnedXP = %d, esperado 500", res.EarnedXP)
		}
		if res.TotalXP != 500 {
			t.Errorf("TotalXP = %d, esperado 500", res.TotalXP)
		}
		if res.NextLevel != exercise.LevelIntermediate || !res.NextUnlocked {
			t.Errorf("esperava desbloqueio de intermediate: %+v", res)
		}

		progress := engine.GetProgress()
		if !progress.IsLevelUnlocked(exercise.LevelIntermediate) {
			t.Errorf("intermediate deveria estar desbloqueado no progresso")
		}
		if progress.CurrentLevel != exercise.LevelIntermediate {
			t.Errorf("CurrentLevel = %s, esperado intermediate", progress.CurrentLevel)
		}
	})

	t.Run("conclusão do nível intermediate desbloqueia advanced", func(t *testing.T) {
		res := engine.CompleteLevel(exercise.LevelIntermediate)
		if res.NextLevel != exercise.LevelAdvanced || !res.NextUnlocked {
			t.Errorf("esperava desbloqueio de advanced: %+v", res)
		}

		progress := engine.GetProgress()
		if !progress.IsLevelUnlocked(exercise.LevelAdvanced) {
			t.Errorf("advanced deveria estar desbloqueado")
		}
		if progress.CurrentLevel != exercise.LevelAdvanced {
			t.Errorf("CurrentLevel = %s, esperado advanced", progress.CurrentLevel)
		}
	})

	t.Run("conclusão do nível advanced finaliza o jogo", func(t *testing.T) {
		res := engine.CompleteLevel(exercise.LevelAdvanced)
		if res.NextLevel != "" || res.NextUnlocked {
			t.Errorf("não deveria haver próximo nível após advanced: %+v", res)
		}
	})
}

func TestEngine_GetProgress_Immutability_And_Reset(t *testing.T) {
	rules := gamification.DefaultRules()
	engine := gamification.NewEngine(rules)

	engine.ProcessAnswer("ex-1", true)
	prog := engine.GetProgress()

	if prog.TotalAttempts != 1 || prog.TotalHits != 1 || prog.BestStreak != 1 {
		t.Errorf("progresso inesperado: %+v", prog)
	}

	// Altera os slices da cópia e verifica que não afeta o estado interno
	prog.UnlockedLevels = append(prog.UnlockedLevels, exercise.LevelAdvanced)
	prog.CompletedLevels = append(prog.CompletedLevels, exercise.LevelBeginner)

	prog2 := engine.GetProgress()
	if len(prog2.UnlockedLevels) != 1 {
		t.Errorf("mutação externa vazou para o estado interno")
	}

	// Reset
	engine.Reset()
	progReset := engine.GetProgress()
	if progReset.TotalXP != 0 || progReset.TotalAttempts != 0 || progReset.CurrentStreak != 0 {
		t.Errorf("esperava progresso zerado após Reset, obteve %+v", progReset)
	}
}
