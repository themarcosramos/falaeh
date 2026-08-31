package exercise_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
)

func TestJSONRepository_Success(t *testing.T) {
	mockFS := fstest.MapFS{
		"beginner.json": &fstest.MapFile{
			Data: []byte(`{
				"level": "beginner",
				"exercises": [
					{
						"id": "beg-1",
						"level": "beginner",
						"type": "multiple_choice",
						"instruction": "Identifique o pato",
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
				"exercises": [
					{
						"id": "adv-1",
						"level": "advanced",
						"type": "sound_identification",
						"instruction": "Qual rima com vulcão?",
						"targetWord": "Vulcão",
						"options": ["Vulcão", "Navio"],
						"correctAnswer": "Vulcão"
					}
				]
			}`),
		},
	}

	repo, err := exercise.NewJSONRepository(mockFS)
	if err != nil {
		t.Fatalf("NewJSONRepository retornou erro inesperado: %v", err)
	}

	// Teste GetByID existente
	ex, err := repo.GetByID("beg-1")
	if err != nil {
		t.Fatalf("GetByID('beg-1') falhou: %v", err)
	}
	if ex.ID != "beg-1" || ex.TargetWord != "Pato" {
		t.Errorf("GetByID retornou dados inconsistentes: %+v", ex)
	}

	// Teste GetByID inexistente
	_, err = repo.GetByID("nao-existe")
	if err == nil {
		t.Errorf("esperava erro ao buscar ID inexistente")
	}

	// Teste GetByLevel
	begList, err := repo.GetByLevel(exercise.LevelBeginner)
	if err != nil {
		t.Fatalf("GetByLevel(beginner) falhou: %v", err)
	}
	if len(begList) != 1 || begList[0].ID != "beg-1" {
		t.Errorf("GetByLevel retornou lista incorreta: %+v", begList)
	}

	// Teste GetByLevel nível inválido
	_, err = repo.GetByLevel(exercise.Level("invalido"))
	if err == nil {
		t.Errorf("esperava erro com nível inválido")
	}

	// Teste ListAll
	all, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll falhou: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("ListAll() retornou tamanho %d, esperado 3", len(all))
	}
}

func TestJSONRepository_Errors(t *testing.T) {
	t.Run("arquivo ausente", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"beginner.json": &fstest.MapFile{Data: []byte(`{"level":"beginner","exercises":[]}`)},
			// intermediate ausente
		}
		_, err := exercise.NewJSONRepository(mockFS)
		if err == nil {
			t.Errorf("esperava erro por arquivo faltante")
		}
	})

	t.Run("json corrompido", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"beginner.json":     &fstest.MapFile{Data: []byte(`{invalido json`)},
			"intermediate.json": &fstest.MapFile{Data: []byte(`{"level":"intermediate","exercises":[]}`)},
			"advanced.json":     &fstest.MapFile{Data: []byte(`{"level":"advanced","exercises":[]}`)},
		}
		_, err := exercise.NewJSONRepository(mockFS)
		if err == nil {
			t.Errorf("esperava erro por json corrompido")
		}
	})

	t.Run("nivel incompativel com arquivo", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"beginner.json":     &fstest.MapFile{Data: []byte(`{"level":"advanced","exercises":[]}`)},
			"intermediate.json": &fstest.MapFile{Data: []byte(`{"level":"intermediate","exercises":[]}`)},
			"advanced.json":     &fstest.MapFile{Data: []byte(`{"level":"advanced","exercises":[]}`)},
		}
		_, err := exercise.NewJSONRepository(mockFS)
		if err == nil {
			t.Errorf("esperava erro por nível declarado incompatível")
		}
	})

	t.Run("exercício inválido no json", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"beginner.json": &fstest.MapFile{Data: []byte(`{
				"level": "beginner",
				"exercises": [{"id": "", "level": "beginner"}]
			}`)},
			"intermediate.json": &fstest.MapFile{Data: []byte(`{"level":"intermediate","exercises":[]}`)},
			"advanced.json":     &fstest.MapFile{Data: []byte(`{"level":"advanced","exercises":[]}`)},
		}
		_, err := exercise.NewJSONRepository(mockFS)
		if err == nil {
			t.Errorf("esperava erro por exercício inválido no arquivo")
		}
	})

	t.Run("exercício com id duplicado", func(t *testing.T) {
		mockFS := fstest.MapFS{
			"beginner.json": &fstest.MapFile{Data: []byte(`{
				"level": "beginner",
				"exercises": [
					{
						"id": "dup-1",
						"level": "beginner",
						"type": "voice",
						"instruction": "Fale pato",
						"targetWord": "Pato",
						"correctAnswer": "Pato"
					},
					{
						"id": "dup-1",
						"level": "beginner",
						"type": "voice",
						"instruction": "Fale gato",
						"targetWord": "Gato",
						"correctAnswer": "Gato"
					}
				]
			}`)},
			"intermediate.json": &fstest.MapFile{Data: []byte(`{"level":"intermediate","exercises":[]}`)},
			"advanced.json":     &fstest.MapFile{Data: []byte(`{"level":"advanced","exercises":[]}`)},
		}
		_, err := exercise.NewJSONRepository(mockFS)
		if err == nil {
			t.Errorf("esperava erro por id duplicado")
		}
	})
}

func TestJSONRepository_WithRealData(t *testing.T) {
	// Carrega do diretório de dados reais de backend/data
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
		t.Fatalf("falha ao carregar dados reais de backend/data: %v", err)
	}

	exercises, err := repo.ListAll()
	if err != nil {
		t.Fatalf("ListAll falhou com dados reais: %v", err)
	}

	if len(exercises) == 0 {
		t.Errorf("esperava exercícios cadastrados nos dados reais")
	}

	for _, ex := range exercises {
		if err := ex.Validate(); err != nil {
			t.Errorf("exercício %s nos dados reais falhou na validação: %v", ex.ID, err)
		}
	}
}
