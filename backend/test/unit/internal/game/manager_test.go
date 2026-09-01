package game_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/game"
	"github.com/themarcosramos/falaeh/backend/internal/gamification"
)

type fakeProvider struct {
	byLevel map[exercise.Level][]exercise.PublicExercise
	answers map[string]string
	listErr error
}

func (f *fakeProvider) ListByLevel(_ context.Context, level exercise.Level) ([]exercise.PublicExercise, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.byLevel[level], nil
}

func (f *fakeProvider) ValidateAnswer(_ context.Context, exerciseID string, answer string) (exercise.ValidationResult, error) {
	expected, ok := f.answers[exerciseID]
	if !ok {
		return exercise.ValidationResult{}, exercise.ErrExerciseNotFound
	}
	return exercise.ValidationResult{
		ExerciseID: exerciseID,
		Correct:    exercise.NormalizeAnswer(answer) == exercise.NormalizeAnswer(expected),
	}, nil
}

func newFakeProvider() *fakeProvider {
	return &fakeProvider{
		byLevel: map[exercise.Level][]exercise.PublicExercise{
			exercise.LevelBeginner: {
				{ID: "beg-001", Level: exercise.LevelBeginner, Type: exercise.TypeMultipleChoice, Options: []string{"Pato", "Gato"}},
				{ID: "beg-002", Level: exercise.LevelBeginner, Type: exercise.TypeMultipleChoice, Options: []string{"Bola", "Mala"}},
			},
			exercise.LevelIntermediate: {
				{ID: "int-001", Level: exercise.LevelIntermediate, Type: exercise.TypeMultipleChoice, Options: []string{"Prato", "Barco"}},
			},
		},
		answers: map[string]string{
			"beg-001": "Pato",
			"beg-002": "Bola",
			"int-001": "Prato",
		},
	}
}

func newManager(t *testing.T, provider game.ExerciseProvider, cfg game.Config) *game.Manager {
	t.Helper()
	return game.NewManager(provider, gamification.DefaultRules(), cfg)
}

func TestManager_Start_CreatesSessionWithFirstExercise(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{})

	result, err := manager.Start(context.Background(), exercise.LevelBeginner, "")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if result.SessionID == "" {
		t.Error("sessionId não deveria ser vazio")
	}
	if result.Exercise.ID != "beg-001" {
		t.Errorf("primeiro exercício = %q, esperado beg-001", result.Exercise.ID)
	}
	if result.TotalExercises != 2 {
		t.Errorf("totalExercises = %d, esperado 2", result.TotalExercises)
	}
	if result.LevelName != "Iniciante" {
		t.Errorf("levelName = %q, esperado Iniciante", result.LevelName)
	}
	if result.Rules.BaseCorrectXP != gamification.DefaultBaseCorrectXP {
		t.Errorf("regras não foram expostas ao cliente: %+v", result.Rules)
	}
	if manager.ActiveSessions() != 1 {
		t.Errorf("sessões ativas = %d, esperado 1", manager.ActiveSessions())
	}
}

func TestManager_Start_InvalidLevel(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{})

	if _, err := manager.Start(context.Background(), exercise.Level("marte"), ""); !errors.Is(err, exercise.ErrInvalidLevel) {
		t.Fatalf("erro = %v, esperado ErrInvalidLevel", err)
	}
}

func TestManager_Start_LockedLevel(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{})

	if _, err := manager.Start(context.Background(), exercise.LevelIntermediate, ""); !errors.Is(err, game.ErrLevelLocked) {
		t.Fatalf("erro = %v, esperado ErrLevelLocked", err)
	}
}

func TestManager_Start_NoExercises(t *testing.T) {
	provider := newFakeProvider()
	provider.byLevel[exercise.LevelBeginner] = nil
	manager := newManager(t, provider, game.Config{})

	if _, err := manager.Start(context.Background(), exercise.LevelBeginner, ""); !errors.Is(err, game.ErrNoExercises) {
		t.Fatalf("erro = %v, esperado ErrNoExercises", err)
	}
}

func TestManager_Start_ProviderError(t *testing.T) {
	provider := newFakeProvider()
	provider.listErr = errors.New("falha de leitura")
	manager := newManager(t, provider, game.Config{})

	if _, err := manager.Start(context.Background(), exercise.LevelBeginner, ""); err == nil {
		t.Fatal("erro esperado ao falhar a leitura dos exercícios")
	}
}

func TestManager_Start_UnknownSessionCreatesNewOne(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{})

	result, err := manager.Start(context.Background(), exercise.LevelBeginner, "sessao-inexistente")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if result.SessionID == "sessao-inexistente" {
		t.Error("uma nova sessão deveria ter sido criada com id próprio")
	}
}

func TestManager_Start_TooManySessions(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{MaxSessions: 1})

	if _, err := manager.Start(context.Background(), exercise.LevelBeginner, ""); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if _, err := manager.Start(context.Background(), exercise.LevelBeginner, ""); !errors.Is(err, game.ErrTooManySessions) {
		t.Fatalf("erro = %v, esperado ErrTooManySessions", err)
	}
}

func TestManager_Answer_CorrectAdvancesAndScores(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{})
	start, err := manager.Start(context.Background(), exercise.LevelBeginner, "")
	if err != nil {
		t.Fatalf("erro inesperado ao iniciar: %v", err)
	}

	result, err := manager.Answer(context.Background(), start.SessionID, "beg-001", "pato")
	if err != nil {
		t.Fatalf("erro inesperado ao responder: %v", err)
	}

	if !result.Correct {
		t.Fatal("resposta deveria ser considerada correta")
	}

	expectedXP := gamification.DefaultBaseCorrectXP + gamification.DefaultFirstAttemptBonusXP
	if result.EarnedXP != expectedXP {
		t.Errorf("earnedXp = %d, esperado %d", result.EarnedXP, expectedXP)
	}
	if result.TotalXP != expectedXP {
		t.Errorf("totalXp = %d, esperado %d", result.TotalXP, expectedXP)
	}
	if result.Streak != 1 {
		t.Errorf("streak = %d, esperado 1", result.Streak)
	}
	if result.NextExercise == nil || result.NextExercise.ID != "beg-002" {
		t.Errorf("nextExercise = %+v, esperado beg-002", result.NextExercise)
	}
	if result.PhaseCompleted {
		t.Error("a fase não deveria estar concluída no primeiro exercício")
	}
}

func TestManager_Answer_WrongKeepsExerciseAndZeroesStreak(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{})
	start, _ := manager.Start(context.Background(), exercise.LevelBeginner, "")

	result, err := manager.Answer(context.Background(), start.SessionID, "beg-001", "gato")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if result.Correct {
		t.Fatal("resposta deveria ser considerada incorreta")
	}
	if result.EarnedXP != 0 || result.TotalXP != 0 {
		t.Errorf("XP = %d/%d, esperado 0/0", result.EarnedXP, result.TotalXP)
	}
	if result.Streak != 0 {
		t.Errorf("streak = %d, esperado 0", result.Streak)
	}
	if result.ExerciseIndex != 0 {
		t.Errorf("exerciseIndex = %d, o jogador deveria permanecer no mesmo exercício", result.ExerciseIndex)
	}
	if result.NextExercise != nil {
		t.Error("nextExercise não deveria ser enviado em resposta incorreta")
	}
}

func TestManager_Answer_SecondAttemptLosesFirstAttemptBonus(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{})
	start, _ := manager.Start(context.Background(), exercise.LevelBeginner, "")

	if _, err := manager.Answer(context.Background(), start.SessionID, "beg-001", "gato"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	result, err := manager.Answer(context.Background(), start.SessionID, "beg-001", "pato")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if result.EarnedXP != gamification.DefaultBaseCorrectXP {
		t.Errorf("earnedXp = %d, esperado %d sem bônus de primeira tentativa", result.EarnedXP, gamification.DefaultBaseCorrectXP)
	}
}

func TestManager_Answer_SessionNotFound(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{})

	if _, err := manager.Answer(context.Background(), "inexistente", "beg-001", "pato"); !errors.Is(err, game.ErrSessionNotFound) {
		t.Fatalf("erro = %v, esperado ErrSessionNotFound", err)
	}
}

func TestManager_Answer_ExerciseMismatch(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{})
	start, _ := manager.Start(context.Background(), exercise.LevelBeginner, "")

	if _, err := manager.Answer(context.Background(), start.SessionID, "beg-002", "bola"); !errors.Is(err, game.ErrExerciseMismatch) {
		t.Fatalf("erro = %v, esperado ErrExerciseMismatch", err)
	}
}

func TestManager_Answer_EmptyExerciseIDUsesCurrent(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{})
	start, _ := manager.Start(context.Background(), exercise.LevelBeginner, "")

	result, err := manager.Answer(context.Background(), start.SessionID, "", "pato")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if result.ExerciseID != "beg-001" {
		t.Errorf("exerciseId = %q, esperado beg-001", result.ExerciseID)
	}
}

func TestManager_CompletePhaseUnlocksNextLevel(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{})
	start, _ := manager.Start(context.Background(), exercise.LevelBeginner, "")

	if _, err := manager.Answer(context.Background(), start.SessionID, "beg-001", "pato"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	final, err := manager.Answer(context.Background(), start.SessionID, "beg-002", "bola")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if !final.PhaseCompleted {
		t.Fatal("a fase deveria estar concluída após o último exercício")
	}
	if final.Completion == nil || !final.Completion.NextUnlocked {
		t.Fatalf("completion = %+v, esperado desbloqueio do próximo nível", final.Completion)
	}
	if final.Completion.NextLevel != exercise.LevelIntermediate {
		t.Errorf("nextLevel = %q, esperado intermediate", final.Completion.NextLevel)
	}
	if final.Report == nil {
		t.Fatal("relatório final não foi gerado")
	}
	if final.Report.Hits != 2 || final.Report.Attempts != 2 || final.Report.Accuracy != 100 {
		t.Errorf("relatório inconsistente: %+v", final.Report)
	}

	// 2 acertos de primeira (300) + bônus de fase (300) + bônus de nível (500)
	expectedXP := 2*(gamification.DefaultBaseCorrectXP+gamification.DefaultFirstAttemptBonusXP) +
		gamification.DefaultPhaseCompletionBonusXP + gamification.DefaultLevelCompletionBonusXP
	if final.TotalXP != expectedXP {
		t.Errorf("totalXp = %d, esperado %d", final.TotalXP, expectedXP)
	}

	if _, err := manager.Answer(context.Background(), start.SessionID, "", "bola"); !errors.Is(err, game.ErrPhaseFinished) {
		t.Fatalf("erro = %v, esperado ErrPhaseFinished", err)
	}
}

func TestManager_ResumeSessionKeepsProgressAcrossLevels(t *testing.T) {
	manager := newManager(t, newFakeProvider(), game.Config{})
	start, _ := manager.Start(context.Background(), exercise.LevelBeginner, "")

	if _, err := manager.Answer(context.Background(), start.SessionID, "beg-001", "pato"); err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	final, err := manager.Answer(context.Background(), start.SessionID, "beg-002", "bola")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	next, err := manager.Start(context.Background(), exercise.LevelIntermediate, start.SessionID)
	if err != nil {
		t.Fatalf("erro inesperado ao iniciar o próximo mundo: %v", err)
	}

	if next.SessionID != start.SessionID {
		t.Errorf("sessionId = %q, esperado reaproveitar %q", next.SessionID, start.SessionID)
	}
	if next.Progress.TotalXP != final.TotalXP {
		t.Errorf("totalXp = %d, esperado preservar %d", next.Progress.TotalXP, final.TotalXP)
	}
	if next.Exercise.ID != "int-001" {
		t.Errorf("exercício = %q, esperado int-001", next.Exercise.ID)
	}
	if manager.ActiveSessions() != 1 {
		t.Errorf("sessões ativas = %d, esperado 1", manager.ActiveSessions())
	}
}

func TestManager_ExpiredSessionIsDiscarded(t *testing.T) {
	current := time.Now()
	manager := newManager(t, newFakeProvider(), game.Config{
		TTL: time.Minute,
		Now: func() time.Time { return current },
	})

	start, err := manager.Start(context.Background(), exercise.LevelBeginner, "")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	current = current.Add(2 * time.Minute)

	if _, err := manager.Answer(context.Background(), start.SessionID, "beg-001", "pato"); !errors.Is(err, game.ErrSessionNotFound) {
		t.Fatalf("erro = %v, esperado ErrSessionNotFound após expiração", err)
	}
	if manager.ActiveSessions() != 0 {
		t.Errorf("sessões ativas = %d, esperado 0 após expiração", manager.ActiveSessions())
	}
}

func TestManager_Answer_ExerciseNotFoundInProvider(t *testing.T) {
	provider := newFakeProvider()
	delete(provider.answers, "beg-001")
	manager := newManager(t, provider, game.Config{})

	start, _ := manager.Start(context.Background(), exercise.LevelBeginner, "")

	if _, err := manager.Answer(context.Background(), start.SessionID, "beg-001", "pato"); !errors.Is(err, exercise.ErrExerciseNotFound) {
		t.Fatalf("erro = %v, esperado ErrExerciseNotFound", err)
	}
}
