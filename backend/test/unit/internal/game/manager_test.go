package game_test

import (
	"context"
	"errors"
	"fmt"
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

func (f *fakeProvider) ValidateAnswer(_ context.Context, exerciseID, answer string) (exercise.ValidationResult, error) {
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
	// Embaralhamento identidade mantém os testes determinísticos; os aspectos
	// de aleatoriedade são cobertos por testes próprios abaixo.
	if cfg.Shuffle == nil {
		cfg.Shuffle = identityShuffle
	}
	return game.NewManager(provider, gamification.DefaultRules(), cfg)
}

func identityShuffle(exercises []exercise.PublicExercise) []exercise.PublicExercise {
	shuffled := make([]exercise.PublicExercise, len(exercises))
	copy(shuffled, exercises)
	return shuffled
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

// reverseShuffle inverte a ordem: ferramenta determinística para provar que a
// apresentação não depende da posição original no banco.
func reverseShuffle(exercises []exercise.PublicExercise) []exercise.PublicExercise {
	shuffled := make([]exercise.PublicExercise, len(exercises))
	for i, ex := range exercises {
		shuffled[len(exercises)-1-i] = ex
	}
	return shuffled
}

func bankProvider(n int) *fakeProvider {
	return multiLevelProvider(map[exercise.Level]int{exercise.LevelBeginner: n})
}

// multiLevelProvider cria bancos com a quantidade informada para cada nível.
func multiLevelProvider(counts map[exercise.Level]int) *fakeProvider {
	prefixes := map[exercise.Level]string{
		exercise.LevelBeginner: "beg", exercise.LevelIntermediate: "int",
		exercise.LevelAdvanced: "adv", exercise.LevelFinal: "fin",
	}
	byLevel := make(map[exercise.Level][]exercise.PublicExercise, len(counts))
	answers := make(map[string]string)
	for level, n := range counts {
		list := make([]exercise.PublicExercise, 0, n)
		for i := 0; i < n; i++ {
			id := fmt.Sprintf("%s-%03d", prefixes[level], i+1)
			list = append(list, exercise.PublicExercise{ID: id, Level: level, Type: exercise.TypeVoice, Instruction: "Pronuncie"})
			answers[id] = id
		}
		byLevel[level] = list
	}
	return &fakeProvider{byLevel: byLevel, answers: answers}
}

// TestManager_FullProgression cobre o fluxo completo: 25 → 35 → 40 acertos desbloqueiam
// os mundos seguintes e o Desafio Final encerra o jogo. Erros no caminho não contam.
func TestManager_FullProgression(t *testing.T) {
	provider := multiLevelProvider(map[exercise.Level]int{
		exercise.LevelBeginner:     gamification.BeginnerTotalExercises,
		exercise.LevelIntermediate: gamification.IntermediateTotalExercises,
		exercise.LevelAdvanced:     gamification.AdvancedTotalExercises,
		exercise.LevelFinal:        gamification.FinalTotalExercises,
	})
	manager := newManager(t, provider, game.Config{})
	ctx := context.Background()
	sessionID := ""

	playPhase := func(level exercise.Level, required int) game.AnswerResult {
		t.Helper()
		start, err := manager.Start(ctx, level, sessionID)
		if err != nil {
			t.Fatalf("erro ao iniciar %s: %v", level, err)
		}
		sessionID = start.SessionID
		if start.TotalExercises != required {
			t.Fatalf("sessão de %s = %d exercícios, esperado %d", level, start.TotalExercises, required)
		}
		current := start.Exercise.ID
		var last game.AnswerResult
		for i := 0; i < required; i++ {
			// Um erro no meio (sem XP, sem progresso) e depois o acerto do mesmo exercício.
			if _, err := manager.Answer(ctx, start.SessionID, current, "errado"); err != nil {
				t.Fatalf("erro ao errar em %s: %v", level, err)
			}
			res, err := manager.Answer(ctx, start.SessionID, current, current)
			if err != nil {
				t.Fatalf("erro ao acertar em %s: %v", level, err)
			}
			last = res
			if i+1 < required {
				if res.NextExercise == nil {
					t.Fatalf("esperava próximo exercício em %s", level)
				}
				current = res.NextExercise.ID
			}
		}
		if !last.PhaseCompleted {
			t.Fatalf("fase %s deveria concluir após %d acertos", level, required)
		}
		return last
	}

	// Mundo 1: bloqueado antes dos 25; com 25 acertos abre o Intermediário.
	if _, err := manager.Start(ctx, exercise.LevelIntermediate, ""); !errors.Is(err, game.ErrLevelLocked) {
		t.Fatalf("intermediate deveria estar bloqueado inicialmente; erro=%v", err)
	}
	if _, err := manager.Start(ctx, exercise.LevelFinal, ""); !errors.Is(err, game.ErrLevelLocked) {
		t.Fatalf("Desafio Final deveria estar bloqueado inicialmente; erro=%v", err)
	}
	beg := playPhase(exercise.LevelBeginner, gamification.BeginnerRequiredHits)
	if beg.Completion == nil || beg.Completion.NextLevel != exercise.LevelIntermediate || !beg.Completion.NextUnlocked {
		t.Fatalf("25 acertos deveriam desbloquear o intermediate: %+v", beg.Completion)
	}

	// Mundo 2: 35 acertos desbloqueiam o Avançado.
	if _, err := manager.Start(ctx, exercise.LevelAdvanced, ""); !errors.Is(err, game.ErrLevelLocked) {
		t.Fatalf("advanced deveria estar bloqueado; erro=%v", err)
	}
	intm := playPhase(exercise.LevelIntermediate, gamification.IntermediateRequiredHits)
	if intm.Completion == nil || intm.Completion.NextLevel != exercise.LevelAdvanced || !intm.Completion.NextUnlocked {
		t.Fatalf("35 acertos deveriam desbloquear o advanced: %+v", intm.Completion)
	}

	// Mundo 3: sessão avançada com 40 exercícios; ao concluir, o Desafio Final abre
	// (não antes: enquanto a fase não terminou, final continua travado).
	start, err := manager.Start(ctx, exercise.LevelAdvanced, sessionID)
	if err != nil {
		t.Fatalf("erro ao iniciar advanced: %v", err)
	}
	sessionID = start.SessionID
	if start.TotalExercises != gamification.AdvancedRequiredHits {
		t.Fatalf("sessão avançada = %d, esperado %d", start.TotalExercises, gamification.AdvancedRequiredHits)
	}
	current := start.Exercise.ID
	var advancedResult game.AnswerResult
	for i := 0; i < gamification.AdvancedRequiredHits; i++ {
		if _, err := manager.Answer(ctx, start.SessionID, current, "errado"); err != nil {
			t.Fatalf("erro ao errar no advanced: %v", err)
		}
		res, err := manager.Answer(ctx, start.SessionID, current, current)
		if err != nil {
			t.Fatalf("erro ao acertar no advanced: %v", err)
		}
		advancedResult = res
		if i+1 < gamification.AdvancedRequiredHits {
			if _, err := manager.Start(ctx, exercise.LevelFinal, ""); !errors.Is(err, game.ErrLevelLocked) {
				t.Fatalf("com %d acertos o Desafio Final deveria continuar travado", i+1)
			}
			current = res.NextExercise.ID
		}
	}
	if _, err := manager.Start(ctx, exercise.LevelFinal, sessionID); err != nil {
		t.Fatalf("com 40 acertos o Desafio Final deveria estar aberto; erro=%v", err)
	}
	if advancedResult.Completion == nil || advancedResult.Completion.NextLevel != exercise.LevelFinal || !advancedResult.Completion.NextUnlocked {
		t.Fatalf("40 acertos deveriam desbloquear o Desafio Final: %+v", advancedResult.Completion)
	}
	if advancedResult.Completion.GameCompleted {
		t.Error("o jogo não deveria estar concluído após o advanced")
	}

	// Desafio Final: 10 acertos concluem o Falaêh.
	fin := playPhase(exercise.LevelFinal, gamification.FinalRequiredHits)
	if fin.Completion == nil || !fin.Completion.GameCompleted {
		t.Fatalf("concluir o Desafio Final deveria encerrar o jogo: %+v", fin.Completion)
	}
	if fin.Completion.NextLevel != "" {
		t.Errorf("não deveria haver próximo nível após o Desafio Final: %+v", fin.Completion)
	}
}

// TestManager_PhaseSequence consumes the session in an orderly manner and verifies the randomized properties.
func TestManager_PhaseSequence_EmbaralhaELimita(t *testing.T) {
	provider := bankProvider(40)
	manager := newManager(t, provider, game.Config{Shuffle: game.DefaultShuffle})

	start, err := manager.Start(context.Background(), exercise.LevelBeginner, "")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}

	if start.TotalExercises != gamification.BeginnerRequiredHits {
		t.Fatalf("sessão devem conter %d exercícios, obteve %d", gamification.BeginnerRequiredHits, start.TotalExercises)
	}
	if start.Exercise.ID == "" || start.Exercise.Level != exercise.LevelBeginner {
		t.Fatalf("exercício fora do mundo iniciante: %+v", start.Exercise)
	}

	// Consome a sessão inteira verificando que nenhum exercício reaparece.
	seen := make(map[string]bool)
	current := start.Exercise.ID
	final := game.AnswerResult{}
	for i := 0; i < start.TotalExercises; i++ {
		if seen[current] {
			t.Fatalf("exercício %q reapareceu na mesma sessão", current)
		}
		seen[current] = true
		res, err := manager.Answer(context.Background(), start.SessionID, current, current)
		if err != nil {
			t.Fatalf("erro na resposta %d: %v", i+1, err)
		}
		if !res.Correct {
			t.Fatalf("resposta %d deveria ser correta", i+1)
		}
		final = res
		if i+1 < start.TotalExercises {
			if res.NextExercise == nil {
				t.Fatalf("esperava próximo exercício na posição %d", i+1)
			}
			current = res.NextExercise.ID
		}
	}

	if !final.PhaseCompleted {
		t.Fatal("a fase deveria concluir com 25 acertos diferentes")
	}
	if final.Report.Hits != gamification.BeginnerRequiredHits {
		t.Errorf("hits = %d, esperado %d", final.Report.Hits, gamification.BeginnerRequiredHits)
	}

	// Sem exercícios repetidos: 25 exercícios distintos, todos do mundo.
	if len(seen) != gamification.BeginnerRequiredHits {
		t.Fatalf("quantidade consumida = %d, esperado %d", len(seen), gamification.BeginnerRequiredHits)
	}
}

// TestManager_AdvancedProgression_40Acertos cobre: erros no caminho não contam como
// acertos; funciona via TestManager_FullProgression, aqui validamos só o banco truncado.
func TestManager_Answer_WrongDoesNotProgress(t *testing.T) {
	provider := bankProvider(gamification.BeginnerRequiredHits + 1)
	manager := newManager(t, provider, game.Config{})

	start, _ := manager.Start(context.Background(), exercise.LevelBeginner, "")
	current := start.Exercise.ID
	var last game.AnswerResult
	for i := 0; i < gamification.BeginnerRequiredHits; i++ {
		if res, err := manager.Answer(context.Background(), start.SessionID, current, "errado"); err != nil {
			t.Fatalf("erro na resposta %d: %v", i+1, err)
		} else {
			if res.Correct {
				t.Fatal("resposta errada não deveria ser aceita")
			}
			if res.ExerciseIndex != i {
				t.Fatalf("índice avançou após erro (%d, esperado %d): tentativas não podem contar como acertos", res.ExerciseIndex, i)
			}
			if res.EarnedXP != 0 {
				t.Errorf("erro concedeu XP inesperado: %d", res.EarnedXP)
			}
		}
		res, err := manager.Answer(context.Background(), start.SessionID, current, current)
		if err != nil {
			t.Fatalf("erro ao acertar o exercício %q: %v", current, err)
		}
		last = res
		if i+1 < gamification.BeginnerRequiredHits {
			current = res.NextExercise.ID
		}
	}

	if !last.PhaseCompleted {
		t.Fatal("após 25 acertos em exercícios diferentes a fase deveria estar concluída")
	}
	if last.Progress.TotalHits != gamification.BeginnerRequiredHits {
		t.Errorf("totalHits = %d, esperado %d", last.Progress.TotalHits, gamification.BeginnerRequiredHits)
	}
}

func TestManager_Start_ShuffleChangesOrder(t *testing.T) {
	provider := bankProvider(12)

	ordered, _ := newManager(t, provider, game.Config{}).Start(context.Background(), exercise.LevelBeginner, "")
	reversed, _ := newManager(t, provider, game.Config{Shuffle: reverseShuffle}).Start(context.Background(), exercise.LevelBeginner, "")

	if ordered.Exercise.ID == reversed.Exercise.ID {
		t.Fatalf("ordens distintas deveriam produzir primeiros exercícios distintos; ambos %q", ordered.Exercise.ID)
	}
	if reversed.TotalExercises != 12 {
		t.Errorf("banco de 12 com exigência 25 deveria preservar todos os 12: %d", reversed.TotalExercises)
	}
}

func TestManager_Start_NewSessionReshuffles(t *testing.T) {
	provider := bankProvider(40)
	// Embaralhador determinístico baseado em um contador: cada nova jogatina
	// rotaciona a entrada, provando que a nova sessão tem nova ordem.
	rotations := 0
	shuffle := func(exs []exercise.PublicExercise) []exercise.PublicExercise {
		rotations++
		out := make([]exercise.PublicExercise, len(exs))
		for i, ex := range exs {
			out[(i+rotations*7)%len(exs)] = ex
		}
		return out
	}
	manager := newManager(t, provider, game.Config{Shuffle: shuffle})

	first, _ := manager.Start(context.Background(), exercise.LevelBeginner, "")
	sid := first.SessionID
	second, _ := manager.Start(context.Background(), exercise.LevelBeginner, sid)
	if second.SessionID != sid {
		t.Fatalf("jogar novamente deveria reaproveitar a sessão: %q != %q", second.SessionID, sid)
	}
	if first.Exercise.ID == second.Exercise.ID {
		t.Fatal("a nova jogatina deveria apresentar nova ordem (primeiro exercício diferente)")
	}
}
