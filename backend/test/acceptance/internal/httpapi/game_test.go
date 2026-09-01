package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/game"
)

func postGameJSON(t *testing.T, router http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("falha ao serializar corpo da requisição: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("falha ao decodificar resposta: %v (corpo: %s)", err, rec.Body.String())
	}

	return out
}

// TestGameFlow_AcertoAvancaExercicio cobre o caminho feliz: iniciar → exercício → responder → XP → próximo.
func TestGameFlow_AcertoAvancaExercicio(t *testing.T) {
	router := newAcceptanceRouter(t)
	repo := acceptanceRepository(t)

	rec := postGameJSON(t, router, "/api/v1/game/start", map[string]string{"level": "beginner"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status do start = %d, esperado %d", rec.Code, http.StatusOK)
	}

	start := decodeBody[game.StartResult](t, rec)
	if start.SessionID == "" || start.Exercise.ID == "" {
		t.Fatalf("start inválido: %+v", start)
	}
	if start.Progress.TotalXP != 0 {
		t.Errorf("totalXp inicial = %d, esperado 0", start.Progress.TotalXP)
	}

	current, err := repo.GetByID(start.Exercise.ID)
	if err != nil {
		t.Fatalf("exercício %q não encontrado no repositório: %v", start.Exercise.ID, err)
	}

	rec = postGameJSON(t, router, "/api/v1/game/"+start.SessionID+"/answer", map[string]string{
		"exerciseId": start.Exercise.ID,
		"answer":     current.CorrectAnswer,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status da resposta = %d, esperado %d", rec.Code, http.StatusOK)
	}

	answer := decodeBody[game.AnswerResult](t, rec)
	if !answer.Correct {
		t.Fatal("resposta correta foi avaliada como incorreta")
	}
	if answer.EarnedXP <= 0 || answer.TotalXP != answer.EarnedXP {
		t.Errorf("XP inconsistente: earned=%d total=%d", answer.EarnedXP, answer.TotalXP)
	}
	if answer.NextExercise == nil {
		t.Fatal("nextExercise deveria ter sido enviado")
	}
	if answer.NextExercise.ID == start.Exercise.ID {
		t.Error("o jogo não avançou para o próximo exercício")
	}
}

// TestGameFlow_ErroPermiteNovaTentativa cobre o cenário de erro sem punição de progresso.
func TestGameFlow_ErroPermiteNovaTentativa(t *testing.T) {
	router := newAcceptanceRouter(t)

	start := decodeBody[game.StartResult](t, postGameJSON(t, router, "/api/v1/game/start", map[string]string{"level": "beginner"}))

	rec := postGameJSON(t, router, "/api/v1/game/"+start.SessionID+"/answer", map[string]string{
		"exerciseId": start.Exercise.ID,
		"answer":     "resposta-definitivamente-errada",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado %d", rec.Code, http.StatusOK)
	}

	answer := decodeBody[game.AnswerResult](t, rec)
	if answer.Correct {
		t.Fatal("resposta errada foi avaliada como correta")
	}
	if answer.TotalXP != 0 {
		t.Errorf("totalXp = %d, esperado 0 após erro", answer.TotalXP)
	}
	if answer.ExerciseIndex != 0 {
		t.Errorf("exerciseIndex = %d, o jogador deveria permanecer no mesmo exercício", answer.ExerciseIndex)
	}
	if answer.NextExercise != nil {
		t.Error("nextExercise não deveria ser enviado após erro")
	}
}

// TestGameFlow_ConclusaoDesbloqueiaProximoNivel percorre todo o nível iniciante até o relatório final.
func TestGameFlow_ConclusaoDesbloqueiaProximoNivel(t *testing.T) {
	router := newAcceptanceRouter(t)
	repo := acceptanceRepository(t)

	start := decodeBody[game.StartResult](t, postGameJSON(t, router, "/api/v1/game/start", map[string]string{"level": "beginner"}))

	currentID := start.Exercise.ID
	var final game.AnswerResult

	for i := 0; i < start.TotalExercises; i++ {
		ex, err := repo.GetByID(currentID)
		if err != nil {
			t.Fatalf("exercício %q não encontrado: %v", currentID, err)
		}

		rec := postGameJSON(t, router, "/api/v1/game/"+start.SessionID+"/answer", map[string]string{
			"exerciseId": currentID,
			"answer":     ex.CorrectAnswer,
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("status na resposta %d = %d", i+1, rec.Code)
		}

		final = decodeBody[game.AnswerResult](t, rec)
		if !final.Correct {
			t.Fatalf("resposta correta do exercício %q foi recusada", currentID)
		}

		if final.NextExercise != nil {
			currentID = final.NextExercise.ID
		}
	}

	if !final.PhaseCompleted {
		t.Fatal("a fase deveria estar concluída ao final do nível")
	}
	if final.Report == nil {
		t.Fatal("relatório final não foi gerado")
	}
	if final.Report.Level != exercise.LevelBeginner || final.Report.Hits != start.TotalExercises {
		t.Errorf("relatório inconsistente: %+v", final.Report)
	}
	if final.Completion == nil || final.Completion.NextLevel != exercise.LevelIntermediate || !final.Completion.NextUnlocked {
		t.Fatalf("desbloqueio do próximo nível não ocorreu: %+v", final.Completion)
	}

	// O nível intermediário só é acessível após a conclusão do iniciante.
	rec := postGameJSON(t, router, "/api/v1/game/start", map[string]string{
		"level":     "intermediate",
		"sessionId": start.SessionID,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("status ao iniciar o nível desbloqueado = %d, esperado %d", rec.Code, http.StatusOK)
	}

	next := decodeBody[game.StartResult](t, rec)
	if next.Progress.TotalXP != final.TotalXP {
		t.Errorf("totalXp = %d, esperado preservar %d", next.Progress.TotalXP, final.TotalXP)
	}
}

// TestGameFlow_NivelBloqueado garante que a progressão não pode ser pulada pelo cliente.
func TestGameFlow_NivelBloqueado(t *testing.T) {
	router := newAcceptanceRouter(t)

	rec := postGameJSON(t, router, "/api/v1/game/start", map[string]string{"level": "advanced"})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, esperado %d", rec.Code, http.StatusForbidden)
	}
}

// TestGameFlow_RespostaNaoVazaGabarito garante que o gabarito nunca é exposto pela API de partida.
func TestGameFlow_RespostaNaoVazaGabarito(t *testing.T) {
	router := newAcceptanceRouter(t)
	repo := acceptanceRepository(t)

	rec := postGameJSON(t, router, "/api/v1/game/start", map[string]string{"level": "beginner"})
	start := decodeBody[game.StartResult](t, rec)

	ex, err := repo.GetByID(start.Exercise.ID)
	if err != nil {
		t.Fatalf("exercício não encontrado: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("falha ao decodificar resposta: %v", err)
	}
	if _, leaked := payload["correctAnswer"]; leaked {
		t.Error("o campo correctAnswer não pode ser exposto")
	}

	rec = postGameJSON(t, router, "/api/v1/game/"+start.SessionID+"/answer", map[string]string{
		"exerciseId": start.Exercise.ID,
		"answer":     "errada",
	})
	if bytes.Contains(rec.Body.Bytes(), []byte(`"correctAnswer"`)) {
		t.Errorf("resposta de erro vazou o gabarito %q: %s", ex.CorrectAnswer, rec.Body.String())
	}
}

// TestGameFlow_PartidaInexistente valida o retorno para sessões desconhecidas ou expiradas.
func TestGameFlow_PartidaInexistente(t *testing.T) {
	router := newAcceptanceRouter(t)

	rec := postGameJSON(t, router, "/api/v1/game/sessao-inexistente/answer", map[string]string{"answer": "pato"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, esperado %d", rec.Code, http.StatusNotFound)
	}
}
