package exercise_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/gamification"
)

// carregaBancoReal abre os arquivos versionados em backend/data.
func carregaBancoReal(t *testing.T) *exercise.JSONRepository {
	t.Helper()
	candidates := []string{"../../../../data", "../../../data", "../../data", "../data", "data"}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "beginner.json")); err == nil {
			dirFS := os.DirFS(c)
			repo, err := exercise.NewJSONRepository(dirFS)
			if err != nil {
				t.Fatalf("falha ao carregar banco de exercícios reais: %v", err)
			}
			return repo
		}
	}
	t.Fatalf("diretório de dados (backend/data) não encontrado em %v", candidates)
	return nil
}

// TestBancoReal_Quantidades valida os critérios da Etapa 19: exatamente
// 40/50/60/10 exercícios, 150 palavras nos mundos e 160 no total.
func TestBancoReal_Quantidades(t *testing.T) {
	repo := carregaBancoReal(t)

	tests := []struct {
		level   exercise.Level
		exactly int
	}{
		{exercise.LevelBeginner, 40},
		{exercise.LevelIntermediate, 50},
		{exercise.LevelAdvanced, 60},
		{exercise.LevelFinal, 10},
	}

	total := 0
	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			list, err := repo.GetByLevel(tt.level)
			if err != nil {
				t.Fatalf("falha ao listar %s: %v", tt.level, err)
			}
			if len(list) != tt.exactly {
				t.Fatalf("mundo %s tem %d exercícios, esperados exatamente %d", tt.level, len(list), tt.exactly)
			}
			for _, ex := range list {
				if err := ex.Validate(); err != nil {
					t.Errorf("exercício %s malformado: %v", ex.ID, err)
				}
				if ex.Level != tt.level {
					t.Errorf("exercício %s declara nível %s, esperado %s", ex.ID, ex.Level, tt.level)
				}
				if words := strings.Fields(ex.TargetWord); len(words) == 0 {
					t.Errorf("exercício %s com palavra vazia", ex.ID)
				}
				// Exercício por voz precisa de opções válidas como fallback de toque.
				if ex.Type == exercise.TypeVoice && len(ex.Options) < 2 {
					t.Errorf("exercício de voz %s deveria oferecer fallback com opções", ex.ID)
				}
			}
			total += len(list)
		})
	}

	if total != 160 {
		t.Errorf("total de exercícios = %d, esperado 160", total)
	}
}

// TestBancoReal_UnicidadeDos150 valida que as 150 palavras dos três mundos são
// únicas (nenhuma repetida dentro do mundo nem entre mundos), usando a mesma
// normalização empregada na validação das respostas.
func TestBancoReal_UnicidadeDos150(t *testing.T) {
	repo := carregaBancoReal(t)

	worlds := map[exercise.Level]struct{}{exercise.LevelBeginner: {}, exercise.LevelIntermediate: {}, exercise.LevelAdvanced: {}}
	unique := make(map[string]exercise.Level)
	total := 0

	for level := range worlds {
		list, err := repo.GetByLevel(level)
		if err != nil {
			t.Fatalf("falha ao listar %s: %v", level, err)
		}
		for _, ex := range list {
			key := exercise.NormalizeAnswer(ex.TargetWord)
			if owner, exists := unique[key]; exists {
				t.Errorf("palavra %q duplicada: %s (%s) e %s (%s)", ex.TargetWord, owner, level, ex.ID, level)
			}
			unique[key] = level
			total++
		}
	}

	if total != 150 {
		t.Errorf("palavras nos três mundos = %d, esperado 150", total)
	}
	if len(unique) != 150 {
		t.Errorf("palavras únicas = %d, esperado 150", len(unique))
	}
}

// TestBancoReal_TodosOsNíveisCarregam garante que o Desafio Final também está no banco real.
func TestBancoReal_TodosOsNíveisCarregam(t *testing.T) {
	repo := carregaBancoReal(t)

	all, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll falhou: %v", err)
	}
	if len(all) != 160 {
		t.Errorf("ListAll = %d exercícios, esperado 160", len(all))
	}

	if _, err := repo.GetByLevel(exercise.LevelFinal); err != nil {
		t.Errorf("Desafio Final deveria estar disponível no banco real: %v", err)
	}
}

// TestBancoReal_CoerenciaDasAlternativas checagens globais de coerência:
// palavras-alvo sempre presentes no exercício e instruções citando a âncora
// quando o alvo não é a própria resposta.
func TestBancoReal_CoerenciaDasAlternativas(t *testing.T) {
	repo := carregaBancoReal(t)

	for _, level := range []exercise.Level{exercise.LevelBeginner, exercise.LevelIntermediate, exercise.LevelAdvanced, exercise.LevelFinal} {
		list, err := repo.GetByLevel(level)
		if err != nil {
			t.Fatalf("falha ao listar %s: %v", level, err)
		}
		for _, ex := range list {
			if !ex.CheckAnswer(ex.CorrectAnswer) {
				t.Errorf("exercício %s: resposta correta não passa pela própria checagem", ex.ID)
			}
			if ex.Type == exercise.TypeMultipleChoice || ex.Type == exercise.TypeImageWordMatch || ex.Type == exercise.TypeSoundIdentification {
				found := false
				for _, opt := range ex.Options {
					if exercise.NormalizeAnswer(opt) == exercise.NormalizeAnswer(ex.CorrectAnswer) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("exercício %s: nenhuma opção corresponde à resposta correta", ex.ID)
				}
			}
		}
	}
}

// TestBancoReal_SeparacaoDisponivelNecessario verifica que as constantes do domínio
// mantêm a separação entre quantidade disponível e acertos necessários.
func TestBancoReal_SeparacaoDisponivelNecessario(t *testing.T) {
	repo := carregaBancoReal(t)

	tests := []struct {
		level     exercise.Level
		available int
		required  int
	}{
		{exercise.LevelBeginner, gamification.BeginnerTotalExercises, gamification.BeginnerRequiredHits},
		{exercise.LevelIntermediate, gamification.IntermediateTotalExercises, gamification.IntermediateRequiredHits},
		{exercise.LevelAdvanced, gamification.AdvancedTotalExercises, gamification.AdvancedRequiredHits},
		{exercise.LevelFinal, gamification.FinalTotalExercises, gamification.FinalRequiredHits},
	}

	for _, tt := range tests {
		list, err := repo.GetByLevel(tt.level)
		if err != nil {
			t.Fatalf("falha ao listar %s: %v", tt.level, err)
		}
		if len(list) != tt.available {
			t.Errorf("%s: banco com %d, esperado %d", tt.level, len(list), tt.available)
		}
		if got := gamification.RequiredHits(tt.level); got != tt.required {
			t.Errorf("%s: acertos necessários %d, esperado %d", tt.level, got, tt.required)
		}
		if tt.available < tt.required {
			t.Errorf("%s: banco (%d) menor que acertos necessários (%d)", tt.level, tt.available, tt.required)
		}
	}
}

// TestBancoReal_RespostaFaladaAceita garante que a palavra que o usuário FALA
// (a palavra-alvo) é aceita como resposta correta em todo exercício cujo alvo
// é a própria resposta (múltipla escolha, imagem/palavra e fala). Nos exercícios
// de identificação sonora a resposta é outro termo (rima/sonoridade), por isso
// são excluídos da igualdade direta.
func TestBancoReal_RespostaFaladaAceita(t *testing.T) {
	repo := carregaBancoReal(t)

	for _, level := range []exercise.Level{
		exercise.LevelBeginner, exercise.LevelIntermediate, exercise.LevelAdvanced, exercise.LevelFinal,
	} {
		list, err := repo.GetByLevel(level)
		if err != nil {
			t.Fatalf("falha ao listar %s: %v", level, err)
		}
		for _, ex := range list {
			if ex.Type == exercise.TypeSoundIdentification {
				continue
			}
			if exercise.NormalizeAnswer(ex.CorrectAnswer) != exercise.NormalizeAnswer(ex.TargetWord) {
				t.Errorf("exercício %s (%s): a resposta falada %q não casa com o gabarito %q após a normalização",
					ex.ID, ex.Type, ex.TargetWord, ex.CorrectAnswer)
			}
		}
	}
}
