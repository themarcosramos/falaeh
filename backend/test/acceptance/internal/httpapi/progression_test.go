package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/game"
	"github.com/themarcosramos/falaeh/backend/internal/gamification"
)

// startFase inicia (ou retoma) uma fase e devolve o estado inicial.
func startFase(t *testing.T, router http.Handler, level exercise.Level, sessionID string) game.StartResult {
	t.Helper()
	rec := postGameJSON(t, router, "/api/v1/game/start", map[string]string{"level": string(level), "sessionId": sessionID})
	if rec.Code != http.StatusOK {
		t.Fatalf("start %s = %d, corpo: %s", level, rec.Code, rec.Body.String())
	}
	return decodeBody[game.StartResult](t, rec)
}

// responderGabarito responde o exercício informado com o gabarito do banco real.
func responderGabarito(t *testing.T, router http.Handler, repo *exercise.JSONRepository, sessionID, exerciseID string) game.AnswerResult {
	t.Helper()
	ex, err := repo.GetByID(exerciseID)
	if err != nil {
		t.Fatalf("exercício %q não encontrado: %v", exerciseID, err)
	}
	rec := postGameJSON(t, router, "/api/v1/game/"+sessionID+"/answer", map[string]string{
		"exerciseId": exerciseID,
		"answer":     ex.CorrectAnswer,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("resposta do exercício %q = %d, corpo: %s", exerciseID, rec.Code, rec.Body.String())
	}
	res := decodeBody[game.AnswerResult](t, rec)
	if !res.Correct {
		t.Fatalf("gabarito do exercício %q foi recusado", exerciseID)
	}
	return res
}

// responderErrado envia uma resposta consistidamente incorreta ao exercício atual.
func responderErrado(t *testing.T, router http.Handler, sessionID, exerciseID string) game.AnswerResult {
	t.Helper()
	rec := postGameJSON(t, router, "/api/v1/game/"+sessionID+"/answer", map[string]string{
		"exerciseId": exerciseID,
		"answer":     "errado-de-propósito-xyz",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("resposta errada do exercício %q = %d", exerciseID, rec.Code)
	}
	res := decodeBody[game.AnswerResult](t, rec)
	if res.Correct {
		t.Fatalf("resposta errada foi aceita para o exercício %q", exerciseID)
	}
	return res
}

// consomeFase responde todos os exercícios da fase (um erro antes de cada acerto)
// e devolve o resultado final da partida junto com o identificador da sessão usada.
func consomeFase(t *testing.T, router http.Handler, repo *exercise.JSONRepository, level exercise.Level, sessionID string) (finalResult game.AnswerResult, finalSessionID string) {
	t.Helper()
	start := startFase(t, router, level, sessionID)

	currentID := start.Exercise.ID
	var last game.AnswerResult
	for i := 0; i < start.TotalExercises; i++ {
		responderErrado(t, router, start.SessionID, currentID)
		last = responderGabarito(t, router, repo, start.SessionID, currentID)
		if i+1 < start.TotalExercises {
			if last.NextExercise == nil {
				t.Fatalf("esperava próximo exercício na posição %d de %s", i+1, level)
			}
			currentID = last.NextExercise.ID
		}
	}
	if !last.PhaseCompleted {
		t.Fatalf("a fase %s deveria concluir com %d acertos", level, start.TotalExercises)
	}
	return last, start.SessionID
}

// TestGameFlow_ProgressaoCompletaMundosEFinal cobre o fluxo completo da jornada:
// Iniciante (25) → Intermediário (35) → Avançado (40) → Desafio Final → Falaeh concluído,
// além de jogar novamente gerando nova sessão com nova ordem.
func TestGameFlow_ProgressaoCompletaMundosEFinal(t *testing.T) {
	router := newAcceptanceRouter(t)
	repo := acceptanceRepository(t)

	// Antes de qualquer progresso: os mundos seguintes e o Desafio Final estão bloqueados.
	for _, locked := range []exercise.Level{exercise.LevelIntermediate, exercise.LevelAdvanced, exercise.LevelFinal} {
		rec := postGameJSON(t, router, "/api/v1/game/start", map[string]string{"level": string(locked)})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s deveria estar bloqueado antes da progressão; status=%d", locked, rec.Code)
		}
	}

	// Mundo 1: 25 acertos em exercícios diferentes liberam o Intermediário.
	beg, sid := consomeFase(t, router, repo, exercise.LevelBeginner, "")
	if beg.Report == nil || beg.Report.Hits != gamification.BeginnerRequiredHits {
		t.Fatalf("relatório do iniciante inconsistente: %+v", beg.Report)
	}
	if beg.Completion == nil || beg.Completion.NextLevel != exercise.LevelIntermediate || !beg.Completion.NextUnlocked {
		t.Fatalf("esperava desbloqueio do intermediate: %+v", beg.Completion)
	}

	// Mundo 2: 35 acertos liberam o Avançado. Com 34, ele permaneceria bloqueado.
	intm, sid := consomeFase(t, router, repo, exercise.LevelIntermediate, sid)
	if intm.Completion == nil || intm.Completion.NextLevel != exercise.LevelAdvanced || !intm.Completion.NextUnlocked {
		t.Fatalf("esperava desbloqueio do advanced: %+v", intm.Completion)
	}

	// Mundo 3: 40 acertos liberam o Desafio Final.
	adv, sid := consomeFase(t, router, repo, exercise.LevelAdvanced, sid)
	if adv.Completion == nil || adv.Completion.NextLevel != exercise.LevelFinal || !adv.Completion.NextUnlocked {
		t.Fatalf("esperava desbloqueio do Desafio Final: %+v", adv.Completion)
	}
	if adv.Completion.GameCompleted {
		t.Error("o jogo não deveria estar concluído após o avançado")
	}

	// Desafio Final: 10 acertos concluem o Falahe.
	fin, sid := consomeFase(t, router, repo, exercise.LevelFinal, sid)
	if fin.Completion == nil || !fin.Completion.GameCompleted {
		t.Fatalf("esperava o fim do jogo após o Desafio Final: %+v", fin.Completion)
	}
	if fin.Completion.NextLevel != "" {
		t.Errorf("não deveria haver próximo nível após o Desafio Final: %+v", fin.Completion)
	}
	if fin.Report == nil {
		t.Fatal("relatório final não foi gerado")
	}

	// Partida concluída recusa novas respostas.
	if rec := postGameJSON(t, router, "/api/v1/game/"+sid+"/answer", map[string]string{"answer": "x"}); rec.Code != http.StatusConflict {
		t.Errorf("partida concluída deveria recusar novas respostas com 409; status=%d", rec.Code)
	}

	// Jogar novamente na mesma sessão preserva XP e reembaralha a ordem.
	firstSeen := make(map[string]bool)
	for i := 0; i < 8; i++ {
		start := startFase(t, router, exercise.LevelBeginner, sid)
		if start.Progress.TotalXP <= 0 {
			t.Error("XP deveria ser preservado ao jogar novamente")
		}
		firstSeen[start.Exercise.ID] = true
	}
	if len(firstSeen) < 2 {
		t.Error("oito novas jogatinas deveriam apresentar primeiros exercícios variados (nova ordem por jogatina)")
	}
}

// TestGameFlow_SemRepeticaoNaSessao percorre a sessão do Iniciante garantindo
// consumo sem reposição: nenhum exercício reaparece na mesma sessão, erros
// não geram XP nem avançam, e a sessão conclui com 25 acertos distintos.
func TestGameFlow_SemRepeticaoNaSessao(t *testing.T) {
	router := newAcceptanceRouter(t)
	repo := acceptanceRepository(t)

	start := startFase(t, router, exercise.LevelBeginner, "")
	if start.TotalExercises != gamification.BeginnerRequiredHits {
		t.Fatalf("sessão iniciante = %d exercícios, esperado %d", start.TotalExercises, gamification.BeginnerRequiredHits)
	}

	seen := make(map[string]bool)
	currentID := start.Exercise.ID
	var last game.AnswerResult
	for i := 0; i < start.TotalExercises; i++ {
		if seen[currentID] {
			t.Fatalf("exercício %q reapareceu antes de concluir a sessão", currentID)
		}
		seen[currentID] = true

		wrong := responderErrado(t, router, start.SessionID, currentID)
		if wrong.EarnedXP != 0 || wrong.ExerciseIndex != i || wrong.NextExercise != nil {
			t.Fatalf("erro deveria manter o jogador no mesmo exercício sem XP: %+v", wrong)
		}

		last = responderGabarito(t, router, repo, start.SessionID, currentID)
		if last.Streak < 1 || last.TotalXP < last.EarnedXP {
			t.Fatalf("acerto inconsistente: %+v", last)
		}
		if i+1 < start.TotalExercises {
			currentID = last.NextExercise.ID
		}
	}

	if len(seen) != start.TotalExercises {
		t.Fatalf("exercícios distintos consumidos = %d, esperado %d", len(seen), start.TotalExercises)
	}
	if !last.PhaseCompleted {
		t.Fatal("a sessão deveria concluir com 25 acertos distintos")
	}

	// Depois de concluída a fase, o exercício concluído não volta: novas respostas são recusadas.
	if rec := postGameJSON(t, router, "/api/v1/game/"+start.SessionID+"/answer", map[string]string{
		"exerciseId": currentID, "answer": "errado-consistente",
	}); rec.Code != http.StatusConflict {
		t.Errorf("sessão concluída deveria recusar respostas com 409; status=%d", rec.Code)
	}
}
