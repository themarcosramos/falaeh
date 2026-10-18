package exercise

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"
)

// Erros sentinela retornados pelo repositório de exercícios.
var (
	ErrExerciseNotFound = errors.New("exercício não encontrado")
	ErrLevelNotFound    = errors.New("nível não encontrado")
)

// Repository define as operações de leitura dos exercícios.
type Repository interface {
	GetByID(id string) (Exercise, error)
	GetByLevel(level Level) ([]Exercise, error)
	ListAll() ([]Exercise, error)
}

type levelDataFile struct {
	Level     Level      `json:"level"`
	Exercises []Exercise `json:"exercises"`
}

// JSONRepository gerencia exercícios carregados a partir de um sistema de arquivos fs.FS.
type JSONRepository struct {
	mu        sync.RWMutex
	exercises map[string]Exercise
	byLevel   map[Level][]Exercise
}

// NewJSONRepository carrega e valida todos os exercícios contidos nos arquivos do fs.FS informado.
func NewJSONRepository(fileSystem fs.FS) (*JSONRepository, error) {
	exercises := make(map[string]Exercise)
	byLevel := make(map[Level][]Exercise)

	levelFiles := []struct {
		level Level
		path  string
	}{
		{level: LevelBeginner, path: "beginner.json"},
		{level: LevelIntermediate, path: "intermediate.json"},
		{level: LevelAdvanced, path: "advanced.json"},
		{level: LevelFinal, path: "final.json"},
	}

	byWord := make(map[string]Level)

	for _, lf := range levelFiles {
		data, err := fs.ReadFile(fileSystem, filepath.Clean(lf.path))
		if err != nil {
			return nil, fmt.Errorf("falha ao ler arquivo de exercícios %s: %w", lf.path, err)
		}

		var fileContent levelDataFile
		if err := json.Unmarshal(data, &fileContent); err != nil {
			return nil, fmt.Errorf("falha ao decodificar json de %s: %w", lf.path, err)
		}

		if fileContent.Level != lf.level {
			return nil, fmt.Errorf("nível declarado no arquivo %s (%s) diverge do esperado (%s)", lf.path, fileContent.Level, lf.level)
		}

		list := make([]Exercise, 0, len(fileContent.Exercises))
		for _, ex := range fileContent.Exercises {
			if err := ex.Validate(); err != nil {
				return nil, fmt.Errorf("exercício inválido %s em %s: %w", ex.ID, lf.path, err)
			}
			if _, exists := exercises[ex.ID]; exists {
				return nil, fmt.Errorf("exercício com id duplicado %s em %s", ex.ID, lf.path)
			}
			if err := requireUniqueWord(ex, byWord, lf.level); err != nil {
				return nil, fmt.Errorf("arquivo %s: %w", lf.path, err)
			}
			exercises[ex.ID] = ex
			list = append(list, ex)
		}

		byLevel[lf.level] = list
	}

	return &JSONRepository{
		exercises: exercises,
		byLevel:   byLevel,
	}, nil
}

// requireUniqueWord garante que cada palavra-alvo pertença a um único exercício:
// nenhuma palavra pode se repetir dentro do próprio nível nem reaparecer em outro.
// A comparação usa a mesma normalização empregada na validação das respostas.
func requireUniqueWord(ex Exercise, byWord map[string]Level, level Level) error {
	key := NormalizeAnswer(ex.TargetWord)
	if owner, exists := byWord[key]; exists {
		if owner == level {
			return fmt.Errorf("palavra %q duplicada dentro do nível %s (exercício %s)", ex.TargetWord, level, ex.ID)
		}
		return fmt.Errorf("palavra %q do nível %s (exercício %s) já pertence ao nível %s", ex.TargetWord, level, ex.ID, owner)
	}
	byWord[key] = level
	return nil
}

// GetByID busca um exercício pelo seu identificador.
func (r *JSONRepository) GetByID(id string) (Exercise, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ex, found := r.exercises[id]
	if !found {
		return Exercise{}, fmt.Errorf("%w: %s", ErrExerciseNotFound, id)
	}

	return ex, nil
}

// GetByLevel retorna os exercícios associados a um determinado nível.
func (r *JSONRepository) GetByLevel(level Level) ([]Exercise, error) {
	if !level.IsValid() {
		return nil, fmt.Errorf("%w: %s", ErrLevelNotFound, level)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	list, ok := r.byLevel[level]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrLevelNotFound, level)
	}

	result := make([]Exercise, len(list))
	copy(result, list)
	return result, nil
}

// ListAll retorna a lista de todos os exercícios registrados em ordem.
func (r *JSONRepository) ListAll() ([]Exercise, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	all := make([]Exercise, 0, len(r.exercises))
	levels := []Level{LevelBeginner, LevelIntermediate, LevelAdvanced, LevelFinal}
	for _, l := range levels {
		if list, ok := r.byLevel[l]; ok {
			all = append(all, list...)
		}
	}

	return all, nil
}
