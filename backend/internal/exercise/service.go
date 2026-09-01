package exercise

import (
	"context"
	"errors"
	"fmt"
)

var ErrEmptyAnswer = errors.New("a resposta enviada não pode ser vazia")

// PublicExercise representa a projeção pública do exercício enviada para o cliente,
// garantindo que a resposta correta não seja vazada prematuramente.
type PublicExercise struct {
	ID          string   `json:"id"`
	Level       Level    `json:"level"`
	Type        Type     `json:"type"`
	Instruction string   `json:"instruction"`
	TargetWord  string   `json:"targetWord,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// ToPublic converte um Exercise em sua projeção segura sem a resposta correta.
func (e Exercise) ToPublic() PublicExercise {
	return PublicExercise{
		ID:          e.ID,
		Level:       e.Level,
		Type:        e.Type,
		Instruction: e.Instruction,
		TargetWord:  e.TargetWord,
		Options:     e.Options,
	}
}

// ValidationResult resume a verificação da resposta enviada pelo usuário.
type ValidationResult struct {
	ExerciseID string `json:"exerciseId"`
	Correct    bool   `json:"correct"`
}

// Service define os casos de uso de exercícios.
type Service struct {
	repo Repository
}

// NewService cria uma nova instância do serviço de exercícios.
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// ListLevels retorna os níveis disponíveis na aplicação.
func (s *Service) ListLevels(_ context.Context) []LevelInfo {
	result := make([]LevelInfo, len(AvailableLevels))
	copy(result, AvailableLevels)
	return result
}

// ListAll lista todos os exercícios de todos os níveis no formato público seguro.
func (s *Service) ListAll(_ context.Context) ([]PublicExercise, error) {
	exercises, err := s.repo.ListAll()
	if err != nil {
		return nil, err
	}

	result := make([]PublicExercise, len(exercises))
	for i, ex := range exercises {
		result[i] = ex.ToPublic()
	}
	return result, nil
}

// GetExerciseByID busca e retorna um exercício por ID no formato público.
func (s *Service) GetExerciseByID(ctx context.Context, id string) (PublicExercise, error) {
	ex, err := s.repo.GetByID(id)
	if err != nil {
		return PublicExercise{}, err
	}
	return ex.ToPublic(), nil
}

// ListByLevel lista os exercícios de um nível no formato público seguro.
func (s *Service) ListByLevel(ctx context.Context, level Level) ([]PublicExercise, error) {
	exercises, err := s.repo.GetByLevel(level)
	if err != nil {
		return nil, err
	}

	result := make([]PublicExercise, len(exercises))
	for i, ex := range exercises {
		result[i] = ex.ToPublic()
	}
	return result, nil
}

// ValidateAnswer avalia a resposta do usuário sem expor dados sensíveis.
func (s *Service) ValidateAnswer(ctx context.Context, exerciseID string, answer string) (ValidationResult, error) {
	if answer == "" {
		return ValidationResult{}, ErrEmptyAnswer
	}

	ex, err := s.repo.GetByID(exerciseID)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("exercício não encontrado: %w", err)
	}

	correct := ex.CheckAnswer(answer)
	return ValidationResult{
		ExerciseID: ex.ID,
		Correct:    correct,
	}, nil
}
