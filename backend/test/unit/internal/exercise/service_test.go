package exercise_test

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
)

func newTestService(t *testing.T) *exercise.Service {
	t.Helper()
	mockFS := fstest.MapFS{
		"beginner.json": &fstest.MapFile{
			Data: []byte(`{
				"level": "beginner",
				"exercises": [
					{
						"id": "beg-1",
						"level": "beginner",
						"type": "multiple_choice",
						"instruction": "Qual palavra começa com P?",
						"targetWord": "Pato",
						"options": ["Pato", "Gato"],
						"correctAnswer": "Pato"
					}
				]
			}`),
		},
		"intermediate.json": &fstest.MapFile{
			Data: []byte(`{
				"level": "intermediate",
				"exercises": [
					{
						"id": "int-1",
						"level": "intermediate",
						"type": "voice",
						"instruction": "Fale dragão",
						"targetWord": "Dragão",
						"correctAnswer": "Dragão"
					}
				]
			}`),
		},
		"advanced.json": &fstest.MapFile{
			Data: []byte(`{
				"level": "advanced",
				"exercises": []
			}`),
		},
	}

	repo, err := exercise.NewJSONRepository(mockFS)
	if err != nil {
		t.Fatalf("erro ao criar repositório mock: %v", err)
	}

	return exercise.NewService(repo)
}

func TestService_GetExerciseByID(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	t.Run("exercício existente é retornado sem vazar resposta correta", func(t *testing.T) {
		pub, err := svc.GetExerciseByID(ctx, "beg-1")
		if err != nil {
			t.Fatalf("GetExerciseByID falhou: %v", err)
		}
		if pub.ID != "beg-1" || pub.TargetWord != "Pato" {
			t.Errorf("dados inesperados do exercício: %+v", pub)
		}
	})

	t.Run("exercício inexistente retorna erro", func(t *testing.T) {
		_, err := svc.GetExerciseByID(ctx, "invalido")
		if err == nil {
			t.Errorf("esperava erro para exercício inexistente")
		}
	})
}

func TestService_ListByLevel(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	t.Run("listar iniciante", func(t *testing.T) {
		list, err := svc.ListByLevel(ctx, exercise.LevelBeginner)
		if err != nil {
			t.Fatalf("ListByLevel falhou: %v", err)
		}
		if len(list) != 1 || list[0].ID != "beg-1" {
			t.Errorf("lista inesperada: %+v", list)
		}
	})

	t.Run("listar nível com erro", func(t *testing.T) {
		_, err := svc.ListByLevel(ctx, exercise.Level("nao-existe"))
		if err == nil {
			t.Errorf("esperava erro para nível inválido")
		}
	})
}

func TestService_ValidateAnswer(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	t.Run("resposta correta normalizada", func(t *testing.T) {
		res, err := svc.ValidateAnswer(ctx, "beg-1", "  pato  ")
		if err != nil {
			t.Fatalf("ValidateAnswer retornou erro: %v", err)
		}
		if !res.Correct {
			t.Errorf("esperava resposta correta, obteve incorreta")
		}
		if res.ExerciseID != "beg-1" {
			t.Errorf("ID do exercício incorreto: %s", res.ExerciseID)
		}
	})

	t.Run("resposta incorreta", func(t *testing.T) {
		res, err := svc.ValidateAnswer(ctx, "beg-1", "Gato")
		if err != nil {
			t.Fatalf("ValidateAnswer retornou erro: %v", err)
		}
		if res.Correct {
			t.Errorf("esperava resposta incorreta")
		}
	})

	t.Run("resposta vazia retorna erro", func(t *testing.T) {
		_, err := svc.ValidateAnswer(ctx, "beg-1", "")
		if err == nil {
			t.Errorf("esperava erro com resposta vazia")
		}
	})

	t.Run("exercício não encontrado retorna erro", func(t *testing.T) {
		_, err := svc.ValidateAnswer(ctx, "inexistente", "Pato")
		if err == nil {
			t.Errorf("esperava erro para exercício inexistente")
		}
	})
}
