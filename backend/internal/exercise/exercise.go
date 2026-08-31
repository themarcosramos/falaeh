package exercise

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Level representa a dificuldade/mundo do exercício.
type Level string

const (
	LevelBeginner     Level = "beginner"
	LevelIntermediate Level = "intermediate"
	LevelAdvanced     Level = "advanced"
)

// IsValid verifica se o nível informado é suportado.
func (l Level) IsValid() bool {
	switch l {
	case LevelBeginner, LevelIntermediate, LevelAdvanced:
		return true
	default:
		return false
	}
}

// Type representa a modalidade de interação do exercício fonoaudiológico.
type Type string

const (
	TypeMultipleChoice      Type = "multiple_choice"
	TypeImageWordMatch      Type = "image_word_match"
	TypeSoundIdentification Type = "sound_identification"
	TypeVoice               Type = "voice"
)

// IsValid verifica se o tipo de exercício é reconhecido pela aplicação.
func (t Type) IsValid() bool {
	switch t {
	case TypeMultipleChoice, TypeImageWordMatch, TypeSoundIdentification, TypeVoice:
		return true
	default:
		return false
	}
}

// Erros de domínio para validação de exercícios.
var (
	ErrInvalidID          = errors.New("id do exercício não pode ser vazio")
	ErrInvalidLevel       = errors.New("nível inválido ou não suportado")
	ErrInvalidType        = errors.New("tipo de exercício inválido ou não suportado")
	ErrEmptyInstruction   = errors.New("instrução do exercício não pode ser vazia")
	ErrEmptyTargetWord    = errors.New("palavra-alvo não pode ser vazia")
	ErrEmptyCorrectAnswer = errors.New("resposta correta não pode ser vazia")
	ErrNotEnoughOptions   = errors.New("exercício deve conter pelo menos duas opções distintas")
	ErrAnswerNotInOptions = errors.New("resposta correta deve estar presente na lista de opções")
)

// Exercise representa uma atividade fonoaudiológica gamificada.
type Exercise struct {
	ID            string   `json:"id"`
	Level         Level    `json:"level"`
	Type          Type     `json:"type"`
	Instruction   string   `json:"instruction"`
	TargetWord    string   `json:"targetWord,omitempty"`
	Options       []string `json:"options,omitempty"`
	CorrectAnswer string   `json:"correctAnswer"`
}

// Validate executa a validação de regras estruturais e de integridade do exercício.
func (e Exercise) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return ErrInvalidID
	}
	if !e.Level.IsValid() {
		return fmt.Errorf("%w: %s", ErrInvalidLevel, e.Level)
	}
	if !e.Type.IsValid() {
		return fmt.Errorf("%w: %s", ErrInvalidType, e.Type)
	}
	if strings.TrimSpace(e.Instruction) == "" {
		return ErrEmptyInstruction
	}
	if strings.TrimSpace(e.TargetWord) == "" {
		return ErrEmptyTargetWord
	}
	if strings.TrimSpace(e.CorrectAnswer) == "" {
		return ErrEmptyCorrectAnswer
	}

	// Tipos de escolha exigem opções válidas
	if e.Type == TypeMultipleChoice || e.Type == TypeImageWordMatch || e.Type == TypeSoundIdentification {
		if len(e.Options) < 2 {
			return ErrNotEnoughOptions
		}

		found := false
		normalizedCorrect := NormalizeAnswer(e.CorrectAnswer)
		for _, opt := range e.Options {
			if NormalizeAnswer(opt) == normalizedCorrect {
				found = true
				break
			}
		}
		if !found {
			return ErrAnswerNotInOptions
		}
	}

	return nil
}

// CheckAnswer avalia se a resposta fornecida pelo usuário está correta segundo a normalização.
func (e Exercise) CheckAnswer(userAnswer string) bool {
	return NormalizeAnswer(userAnswer) == NormalizeAnswer(e.CorrectAnswer)
}

// NormalizeAnswer padroniza o texto removendo espaços periféricos, caracteres de controle
// e tratando pontuação para uma comparação justa de voz e texto.
func NormalizeAnswer(s string) string {
	var sb strings.Builder
	for _, r := range strings.TrimSpace(strings.ToLower(s)) {
		if unicode.IsPunct(r) && r != '-' {
			continue
		}
		sb.WriteRune(r)
	}
	return strings.Join(strings.Fields(sb.String()), " ")
}
