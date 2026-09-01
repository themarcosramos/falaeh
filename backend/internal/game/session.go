package game

import (
	"errors"
	"sync"
	"time"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/gamification"
)

// Erros de domínio da partida.
var (
	ErrSessionNotFound  = errors.New("partida não encontrada ou expirada")
	ErrLevelLocked      = errors.New("nível ainda não desbloqueado")
	ErrNoExercises      = errors.New("nível não possui exercícios disponíveis")
	ErrExerciseMismatch = errors.New("exercício informado não é o exercício atual da partida")
	ErrPhaseFinished    = errors.New("a fase já foi concluída")
	ErrTooManySessions  = errors.New("limite de partidas simultâneas atingido")
)

// StartResult descreve o estado inicial de uma fase recém-iniciada.
type StartResult struct {
	SessionID      string                  `json:"sessionId"`
	Level          exercise.Level          `json:"level"`
	LevelName      string                  `json:"levelName"`
	ExerciseIndex  int                     `json:"exerciseIndex"`
	TotalExercises int                     `json:"totalExercises"`
	Exercise       exercise.PublicExercise `json:"exercise"`
	Progress       gamification.Progress   `json:"progress"`
	Rules          gamification.Rules      `json:"rules"`
}

// AnswerResult descreve o efeito de uma resposta sobre a partida.
type AnswerResult struct {
	ExerciseID      string                              `json:"exerciseId"`
	Correct         bool                                `json:"correct"`
	EarnedXP        int                                 `json:"earnedXp"`
	Breakdown       gamification.XPBreakdown            `json:"breakdown"`
	TotalXP         int                                 `json:"totalXp"`
	Streak          int                                 `json:"streak"`
	NewAchievements []gamification.Achievement          `json:"newAchievements,omitempty"`
	ExerciseIndex   int                                 `json:"exerciseIndex"`
	TotalExercises  int                                 `json:"totalExercises"`
	NextExercise    *exercise.PublicExercise            `json:"nextExercise,omitempty"`
	PhaseCompleted  bool                                `json:"phaseCompleted"`
	Completion      *gamification.LevelCompletionResult `json:"completion,omitempty"`
	Report          *Report                             `json:"report,omitempty"`
	Progress        gamification.Progress               `json:"progress"`
}

// Report resume a partida para a tela de resultado final gerada pelo frontend.
type Report struct {
	Level             exercise.Level `json:"level"`
	LevelName         string         `json:"levelName"`
	ExercisesTotal    int            `json:"exercisesTotal"`
	Hits              int            `json:"hits"`
	Misses            int            `json:"misses"`
	Attempts          int            `json:"attempts"`
	Accuracy          int            `json:"accuracy"`
	TotalXP           int            `json:"totalXp"`
	BestStreak        int            `json:"bestStreak"`
	PhaseCompleted    bool           `json:"phaseCompleted"`
	NextLevel         exercise.Level `json:"nextLevel,omitempty"`
	NextLevelUnlocked bool           `json:"nextLevelUnlocked"`
}

// session mantém o estado de uma partida durante a sessão do jogador.
// O estado vive apenas em memória: nada é persistido nem associado a dados pessoais.
type session struct {
	mu         sync.Mutex
	id         string
	level      exercise.Level
	exercises  []exercise.PublicExercise
	index      int
	completed  bool
	engine     *gamification.Engine
	lastSeenAt time.Time
}

// startPhase reinicia a sessão para uma nova fase preservando XP, conquistas e desbloqueios.
func (s *session) startPhase(level exercise.Level, exercises []exercise.PublicExercise, now time.Time) {
	s.level = level
	s.exercises = exercises
	s.index = 0
	s.completed = false
	s.lastSeenAt = now
}

func (s *session) currentExercise() (exercise.PublicExercise, bool) {
	if s.index < 0 || s.index >= len(s.exercises) {
		return exercise.PublicExercise{}, false
	}
	return s.exercises[s.index], true
}

func (s *session) buildReport(completion gamification.LevelCompletionResult, progress gamification.Progress) Report {
	accuracy := 0
	if progress.TotalAttempts > 0 {
		accuracy = progress.TotalHits * 100 / progress.TotalAttempts
	}

	return Report{
		Level:             s.level,
		LevelName:         levelName(s.level),
		ExercisesTotal:    len(s.exercises),
		Hits:              progress.TotalHits,
		Misses:            progress.TotalMisses,
		Attempts:          progress.TotalAttempts,
		Accuracy:          accuracy,
		TotalXP:           progress.TotalXP,
		BestStreak:        progress.BestStreak,
		PhaseCompleted:    true,
		NextLevel:         completion.NextLevel,
		NextLevelUnlocked: completion.NextUnlocked,
	}
}

func levelName(level exercise.Level) string {
	for _, info := range exercise.AvailableLevels {
		if info.ID == level {
			return info.Name
		}
	}
	return string(level)
}
