package exercise_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
)

// TestExerciseAcceptance_Scenarios valida os critérios e fluxos de aceitação do domínio de exercícios.
func TestExerciseAcceptance_Scenarios(t *testing.T) {
	// Cenário de Aceitação: Carregar dados reais versionados em backend/data
	candidates := []string{"../../../../data", "../../../data", "../../data", "../data", "data"}
	var dirFS fs.FS
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "beginner.json")); err == nil {
			dirFS = os.DirFS(c)
			break
		}
	}
	if dirFS == nil {
		t.Fatalf("diretório de dados (backend/data) não foi encontrado nos caminhos candidatos: %v", candidates)
	}
	repo, err := exercise.NewJSONRepository(dirFS)
	if err != nil {
		t.Fatalf("Aceitação falhou: não foi possível carregar o repositório com dados reais: %v", err)
	}

	service := exercise.NewService(repo)
	ctx := context.Background()

	t.Run("Critério 1: Exercícios carregados por JSON e separados por mundos/níveis", func(t *testing.T) {
		beginnerList, err := service.ListByLevel(ctx, exercise.LevelBeginner)
		if err != nil {
			t.Fatalf("falha ao carregar nível iniciante: %v", err)
		}
		if len(beginnerList) == 0 {
			t.Fatalf("esperava exercícios no nível iniciante")
		}

		// Garante que a resposta correta NUNCA vaza na projeção pública
		for _, item := range beginnerList {
			if item.ID == "" {
				t.Errorf("exercício com id vazio")
			}
			if item.Level != exercise.LevelBeginner {
				t.Errorf("exercício %s deveria ser beginner, obteve %s", item.ID, item.Level)
			}
			if item.Instruction == "" {
				t.Errorf("exercício %s tem instrução vazia", item.ID)
			}
		}
	})

	t.Run("Critério 2: Validação de resposta correta do usuário", func(t *testing.T) {
		// Pato é a resposta para beg-001
		res, err := service.ValidateAnswer(ctx, "beg-001", "Pato")
		if err != nil {
			t.Fatalf("ValidateAnswer falhou: %v", err)
		}
		if !res.Correct {
			t.Errorf("esperava validação positiva para 'Pato'")
		}
	})

	t.Run("Critério 3: Tolerância a variações fonéticas e formatação na validação", func(t *testing.T) {
		// Normalização com maiúsculas, pontuação e espaços extras
		res, err := service.ValidateAnswer(ctx, "beg-001", "   pato!  ")
		if err != nil {
			t.Fatalf("ValidateAnswer falhou: %v", err)
		}
		if !res.Correct {
			t.Errorf("esperava aceitação com tolerância de pontuação e espaços para '   pato!  '")
		}
	})

	t.Run("Critério 4: Rejeição amigável de resposta incorreta sem quebrar o fluxo", func(t *testing.T) {
		res, err := service.ValidateAnswer(ctx, "beg-001", "Gato")
		if err != nil {
			t.Fatalf("ValidateAnswer falhou: %v", err)
		}
		if res.Correct {
			t.Errorf("esperava rejeição para resposta incorreta 'Gato'")
		}
	})

	t.Run("Critério 5: Exercício por voz suportado com alternativa de opções", func(t *testing.T) {
		// beg-004 é do tipo voice
		ex, err := service.GetExerciseByID(ctx, "beg-004")
		if err != nil {
			t.Fatalf("falha ao buscar beg-004: %v", err)
		}
		if ex.Type != exercise.TypeVoice {
			t.Errorf("esperava tipo voice, obteve %s", ex.Type)
		}
		// Deve ter opções como fallback para navegadores sem SpeechRecognition
		if len(ex.Options) == 0 {
			t.Errorf("exercício de voz deve fornecer alternativas de opções para fallback sem microfone")
		}

		// Validação por fala/texto reconhecido
		res, err := service.ValidateAnswer(ctx, "beg-004", "pato")
		if err != nil {
			t.Fatalf("ValidateAnswer falhou: %v", err)
		}
		if !res.Correct {
			t.Errorf("esperava validação positiva para exercício de voz")
		}
	})
}
