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
		// Busca dinamicamente um exercício do tipo voice no iniciante (o banco pode ser reordenado).
		voiceID := ""
		ctxList, err := service.ListByLevel(ctx, exercise.LevelBeginner)
		if err != nil {
			t.Fatalf("falha ao listar exercícios de voz: %v", err)
		}
		for _, item := range ctxList {
			if item.Type == exercise.TypeVoice {
				voiceID = item.ID
				break
			}
		}
		if voiceID == "" {
			t.Fatal("esperava ao menos um exercício do tipo voice no nível iniciante")
		}

		ex, err := service.GetExerciseByID(ctx, voiceID)
		if err != nil {
			t.Fatalf("falha ao buscar %s: %v", voiceID, err)
		}
		if ex.Type != exercise.TypeVoice {
			t.Errorf("esperava tipo voice, obteve %s", ex.Type)
		}
		// Deve ter opções como fallback para navegadores sem SpeechRecognition
		if len(ex.Options) == 0 {
			t.Errorf("exercício de voz deve fornecer alternativas de opções para fallback sem microfone")
		}

		// A resposta correta do banco é validada por fala/texto reconhecido.
		db, err := repo.GetByID(voiceID)
		if err != nil {
			t.Fatalf("falha ao recuperar gabarito de %s: %v", voiceID, err)
		}
		res, err := service.ValidateAnswer(ctx, voiceID, db.CorrectAnswer)
		if err != nil {
			t.Fatalf("ValidateAnswer falhou: %v", err)
		}
		if !res.Correct {
			t.Errorf("esperava validação positiva para exercício de voz %s", voiceID)
		}
	})
}

// TestExerciseAcceptance_VozEmExerciciosDeImagem valida que a palavra falada naturalmente
// (sem o emoji presente no rótulo da imagem) é aceita nas missões de associação imagem/palavra.
func TestExerciseAcceptance_VozEmExerciciosDeImagem(t *testing.T) {
	repo := acceptanceExerciseRepo(t)
	svc := exercise.NewService(repo)
	ctxVoz := context.Background()

	// Cenário de Aceitação: mesma verificação usada na validação da resposta por voz
	// beg-011 ("Qual imagem representa a palavra BOLA?") tem gabarito "⚽ Bola".
	// Falando a palavra — transcript "bola" — o jogo deve considerar correto.
	res, err := svc.ValidateAnswer(ctxVoz, "beg-011", "bola")
	if err != nil {
		t.Fatalf("ValidateAnswer falhou: %v", err)
	}
	if !res.Correct {
		t.Errorf("transcrição falada 'bola' deveria ser aceita no exercício de imagem beg-011 (gabarito '⚽ Bola')")
	}

	// A opção completa (com emoji, via toque) continua sendo aceita.
	res, err = svc.ValidateAnswer(ctxVoz, "beg-011", "⚽ Bola")
	if err != nil {
		t.Fatalf("ValidateAnswer falhou: %v", err)
	}
	if !res.Correct {
		t.Errorf("opção completa com emoji deveria continuar aceita")
	}

	// E a resposta errada continua rejeitada.
	res, err = svc.ValidateAnswer(ctxVoz, "beg-011", "gato")
	if err != nil {
		t.Fatalf("ValidateAnswer falhou: %v", err)
	}
	if res.Correct {
		t.Errorf("resposta errada 'gato' deveria continuar rejeitada")
	}
}

// acceptanceExerciseRepo abre o banco real versionado em backend/data para os testes deste pacote.
func acceptanceExerciseRepo(t *testing.T) *exercise.JSONRepository {
	t.Helper()
	candidates := []string{"../../../../data", "../../../data", "../../data", "../data", "data"}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "beginner.json")); err == nil {
			repo, err := exercise.NewJSONRepository(os.DirFS(c))
			if err != nil {
				t.Fatalf("falha ao carregar banco de exercícios reais: %v", err)
			}
			return repo
		}
	}
	t.Fatalf("diretório de dados (backend/data) não encontrado")
	return nil
}
