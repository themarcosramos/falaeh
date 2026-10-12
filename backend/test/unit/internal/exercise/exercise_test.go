package exercise_test

import (
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
)

func TestLevel_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		level exercise.Level
		want  bool
	}{
		{"beginner", exercise.LevelBeginner, true},
		{"intermediate", exercise.LevelIntermediate, true},
		{"advanced", exercise.LevelAdvanced, true},
		{"invalid", exercise.Level("expert"), false},
		{"empty", exercise.Level(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.level.IsValid(); got != tt.want {
				t.Errorf("Level.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		itemType exercise.Type
		want     bool
	}{
		{"multiple_choice", exercise.TypeMultipleChoice, true},
		{"image_word_match", exercise.TypeImageWordMatch, true},
		{"sound_identification", exercise.TypeSoundIdentification, true},
		{"voice", exercise.TypeVoice, true},
		{"unknown", exercise.Type("unknown"), false},
		{"empty", exercise.Type(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.itemType.IsValid(); got != tt.want {
				t.Errorf("Type.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExercise_Validate(t *testing.T) {
	validMC := exercise.Exercise{
		ID:            "beg-001",
		Level:         exercise.LevelBeginner,
		Type:          exercise.TypeMultipleChoice,
		Instruction:   "Selecione a opção",
		TargetWord:    "Pato",
		Options:       []string{"Pato", "Gato"},
		CorrectAnswer: "Pato",
	}

	tests := []struct {
		name    string
		mutate  func(e *exercise.Exercise)
		wantErr bool
	}{
		{
			name:    "exercício válido",
			mutate:  func(e *exercise.Exercise) {},
			wantErr: false,
		},
		{
			name: "id vazio",
			mutate: func(e *exercise.Exercise) {
				e.ID = "   "
			},
			wantErr: true,
		},
		{
			name: "nível inválido",
			mutate: func(e *exercise.Exercise) {
				e.Level = "invalid"
			},
			wantErr: true,
		},
		{
			name: "tipo inválido",
			mutate: func(e *exercise.Exercise) {
				e.Type = "invalid"
			},
			wantErr: true,
		},
		{
			name: "instrução vazia",
			mutate: func(e *exercise.Exercise) {
				e.Instruction = " "
			},
			wantErr: true,
		},
		{
			name: "target word vazia",
			mutate: func(e *exercise.Exercise) {
				e.TargetWord = ""
			},
			wantErr: true,
		},
		{
			name: "resposta correta vazia",
			mutate: func(e *exercise.Exercise) {
				e.CorrectAnswer = " "
			},
			wantErr: true,
		},
		{
			name: "múltipla escolha com menos de 2 opções",
			mutate: func(e *exercise.Exercise) {
				e.Options = []string{"Pato"}
			},
			wantErr: true,
		},
		{
			name: "resposta correta ausente das opções",
			mutate: func(e *exercise.Exercise) {
				e.Options = []string{"Gato", "Cão"}
			},
			wantErr: true,
		},
		{
			name: "exercício de voz sem opções é válido",
			mutate: func(e *exercise.Exercise) {
				e.Type = exercise.TypeVoice
				e.Options = nil
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ex := validMC
			tt.mutate(&ex)
			err := ex.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Exercise.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExercise_CheckAnswer_And_Normalize(t *testing.T) {
	tests := []struct {
		name          string
		correctAnswer string
		userAnswer    string
		want          bool
	}{
		{"exato", "Pato", "Pato", true},
		{"diferença de maiúsculas/minúsculas", "Pato", "pato", true},
		{"espaços extras no início e fim", "Bola", "  bola  ", true},
		{"espaços múltiplos intermediários", "O sapo no saco", "o   sapo   no   saco", true},
		{"pontuação removida na checagem", "Pão!", "pão", true},
		{"interrogação e ponto final", "Quem é?", "quem é.", true},
		{"resposta incorreta", "Pato", "Gato", false},
		{"resposta vazia", "Pato", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ex := exercise.Exercise{
				CorrectAnswer: tt.correctAnswer,
			}
			got := ex.CheckAnswer(tt.userAnswer)
			if got != tt.want {
				t.Errorf("CheckAnswer(%q, %q) = %v, want %v", tt.correctAnswer, tt.userAnswer, got, tt.want)
			}
		})
	}
}
