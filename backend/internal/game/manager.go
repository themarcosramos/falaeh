package game

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/gamification"
)

const (
	defaultSessionTTL  = 2 * time.Hour
	defaultMaxSessions = 5000
	sessionIDBytes     = 16
)

// ExerciseProvider descreve o que a partida precisa do catálogo de exercícios.
type ExerciseProvider interface {
	ListByLevel(ctx context.Context, level exercise.Level) ([]exercise.PublicExercise, error)
	ValidateAnswer(ctx context.Context, exerciseID string, answer string) (exercise.ValidationResult, error)
}

// Config ajusta os limites do gerenciador de partidas. O zero-value usa os padrões.
type Config struct {
	TTL         time.Duration
	MaxSessions int
	Now         func() time.Time
}

// Manager guarda as partidas em andamento apenas em memória, com expiração por inatividade.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session

	exercises   ExerciseProvider
	rules       gamification.Rules
	ttl         time.Duration
	maxSessions int
	now         func() time.Time
}

// NewManager cria o gerenciador de partidas do jogo.
func NewManager(exercises ExerciseProvider, rules gamification.Rules, cfg Config) *Manager {
	if cfg.TTL <= 0 {
		cfg.TTL = defaultSessionTTL
	}
	if cfg.MaxSessions <= 0 {
		cfg.MaxSessions = defaultMaxSessions
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	return &Manager{
		sessions:    make(map[string]*session),
		exercises:   exercises,
		rules:       rules,
		ttl:         cfg.TTL,
		maxSessions: cfg.MaxSessions,
		now:         cfg.Now,
	}
}

// Start inicia uma fase. Informar um sessionID existente preserva XP, conquistas
// e níveis desbloqueados; caso contrário uma nova partida é criada.
func (m *Manager) Start(ctx context.Context, level exercise.Level, sessionID string) (StartResult, error) {
	if !level.IsValid() {
		return StartResult{}, fmt.Errorf("%w: %s", exercise.ErrInvalidLevel, level)
	}

	exercises, err := m.exercises.ListByLevel(ctx, level)
	if err != nil {
		return StartResult{}, err
	}
	if len(exercises) == 0 {
		return StartResult{}, ErrNoExercises
	}

	sess, err := m.resolveSession(sessionID)
	if err != nil {
		return StartResult{}, err
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	progress := sess.engine.GetProgress()
	if !progress.IsLevelUnlocked(level) {
		return StartResult{}, ErrLevelLocked
	}

	sess.startPhase(level, exercises, m.now())
	current, _ := sess.currentExercise()

	return StartResult{
		SessionID:      sess.id,
		Level:          level,
		LevelName:      levelName(level),
		ExerciseIndex:  sess.index,
		TotalExercises: len(sess.exercises),
		Exercise:       current,
		Progress:       progress,
		Rules:          m.rules,
	}, nil
}

// Answer valida a resposta do exercício atual e devolve o resultado com XP,
// streak e o próximo exercício. Toda a pontuação é decidida aqui, nunca no cliente.
func (m *Manager) Answer(ctx context.Context, sessionID, exerciseID, answer string) (AnswerResult, error) {
	sess, ok := m.lookup(sessionID)
	if !ok {
		return AnswerResult{}, ErrSessionNotFound
	}

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if sess.completed {
		return AnswerResult{}, ErrPhaseFinished
	}

	current, ok := sess.currentExercise()
	if !ok {
		return AnswerResult{}, ErrPhaseFinished
	}
	if exerciseID != "" && exerciseID != current.ID {
		return AnswerResult{}, ErrExerciseMismatch
	}

	validation, err := m.exercises.ValidateAnswer(ctx, current.ID, answer)
	if err != nil {
		return AnswerResult{}, err
	}

	sess.lastSeenAt = m.now()
	attempt := sess.engine.ProcessAnswer(current.ID, validation.Correct)

	result := AnswerResult{
		ExerciseID:      current.ID,
		Correct:         attempt.Correct,
		EarnedXP:        attempt.EarnedXP,
		Breakdown:       attempt.Breakdown,
		Streak:          attempt.Streak,
		NewAchievements: attempt.NewAchievements,
		TotalExercises:  len(sess.exercises),
	}

	if attempt.Correct {
		sess.index++

		if sess.index >= len(sess.exercises) {
			sess.completed = true
			sess.engine.CompletePhase(string(sess.level))
			completion := sess.engine.CompleteLevel(sess.level)
			progress := sess.engine.GetProgress()
			report := sess.buildReport(completion, progress)

			result.PhaseCompleted = true
			result.Completion = &completion
			result.Report = &report
			result.NewAchievements = append(result.NewAchievements, completion.NewAchievements...)
		} else {
			next := sess.exercises[sess.index]
			result.NextExercise = &next
		}
	}

	result.ExerciseIndex = sess.index
	result.Progress = sess.engine.GetProgress()
	result.TotalXP = result.Progress.TotalXP

	return result, nil
}

// resolveSession reaproveita uma partida válida ou cria uma nova.
func (m *Manager) resolveSession(sessionID string) (*session, error) {
	if sessionID != "" {
		if sess, ok := m.lookup(sessionID); ok {
			return sess, nil
		}
	}

	id, err := newSessionID()
	if err != nil {
		return nil, err
	}

	sess := &session{
		id:         id,
		engine:     gamification.NewEngine(m.rules),
		lastSeenAt: m.now(),
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.evictExpiredLocked()
	if len(m.sessions) >= m.maxSessions {
		return nil, ErrTooManySessions
	}
	m.sessions[id] = sess

	return sess, nil
}

func (m *Manager) lookup(sessionID string) (*session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[sessionID]
	if !ok {
		return nil, false
	}

	if m.now().Sub(sess.lastSeenAt) > m.ttl {
		delete(m.sessions, sessionID)
		return nil, false
	}

	return sess, true
}

func (m *Manager) evictExpiredLocked() {
	deadline := m.now().Add(-m.ttl)
	for id, sess := range m.sessions {
		if sess.lastSeenAt.Before(deadline) {
			delete(m.sessions, id)
		}
	}
}

// ActiveSessions informa quantas partidas estão em memória (usado em observabilidade e testes).
func (m *Manager) ActiveSessions() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

func newSessionID() (string, error) {
	buf := make([]byte, sessionIDBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("falha ao gerar identificador de partida: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
