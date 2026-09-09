// Package feedback gerencia avaliações da experiência do Fala Eh.
// Todas as avaliações são anônimas, fechadas e totalmente desacopladas dos dados da partida.
package feedback

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Rating representa uma opção fechada de avaliação da experiência.
type Rating string

// Opções válidas de avaliação da experiência pelo usuário.
const (
	RatingLoved    Rating = "gostei_muito"
	RatingLiked    Rating = "gostei"
	RatingNeutral  Rating = "mais_ou_menos"
	RatingDisliked Rating = "nao_gostei"

	// MaxCommentLength limita o tamanho do comentário descritivo opcional (200 caracteres).
	MaxCommentLength = 200
)

// Erros sentinela retornados na validação de feedback.
var (
	ErrInvalidRating = errors.New("avaliação inválida: opções permitidas são gostei_muito, gostei, mais_ou_menos ou nao_gostei")
	ErrEmptyRating   = errors.New("a avaliação não pode ser vazia")
)

// Feedback representa uma avaliação anônima contendo a nota fechada e um comentário opcional.
type Feedback struct {
	Rating  Rating
	Comment string
}

// IsValid verifica se a opção de avaliação pertence ao conjunto fechado de respostas permitidas.
func (r Rating) IsValid() bool {
	switch r {
	case RatingLoved, RatingLiked, RatingNeutral, RatingDisliked:
		return true
	default:
		return false
	}
}

// Description retorna o ícone e a descrição em português da avaliação.
func (r Rating) Description() string {
	switch r {
	case RatingLoved:
		return "😄 Gostei muito"
	case RatingLiked:
		return "🙂 Gostei"
	case RatingNeutral:
		return "😐 Foi mais ou menos"
	case RatingDisliked:
		return "🙁 Não gostei"
	default:
		return string(r)
	}
}

// ParseRating normaliza e valida o valor bruto recebido.
func ParseRating(value string) (Rating, error) {
	clean := strings.ToLower(strings.TrimSpace(value))
	if clean == "" {
		return "", ErrEmptyRating
	}

	r := Rating(clean)
	if !r.IsValid() {
		return "", ErrInvalidRating
	}

	return r, nil
}

// SanitizeComment limpa e limita o tamanho do comentário opcional.
func SanitizeComment(raw string) string {
	trimmed := strings.TrimSpace(raw)
	runes := []rune(trimmed)
	if len(runes) > MaxCommentLength {
		runes = runes[:MaxCommentLength]
	}
	return string(runes)
}

// Sender é o contrato para envio da avaliação para um provedor externo opcional (ex: Google Forms).
type Sender interface {
	Send(ctx context.Context, fb Feedback) error
}

// Service orquestra a validação e encaminhamento da avaliação anônima.
// Não cria persistência própria nem armazena identificadores ou dados pessoais.
type Service struct {
	sender Sender
	logger *slog.Logger
}

// NewService instancia o serviço de avaliação.
func NewService(sender Sender, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		sender: sender,
		logger: logger,
	}
}

// Submit valida a opção fechada e a encaminha para o sender opcional.
func (s *Service) Submit(ctx context.Context, rawRating, rawComment string) error {
	rating, err := ParseRating(rawRating)
	if err != nil {
		return err
	}

	comment := SanitizeComment(rawComment)
	fb := Feedback{
		Rating:  rating,
		Comment: comment,
	}

	s.logger.Info("avaliação anônima recebida",
		slog.String("rating", string(rating)),
		slog.Bool("has_comment", comment != ""),
	)

	if s.sender != nil {
		if err := s.sender.Send(ctx, fb); err != nil {
			s.logger.Warn("falha ao encaminhar avaliação para o provedor externo", slog.Any("error", err))
			// Falhas no provedor externo nunca impedem o encerramento normal
			return err
		}
	}

	return nil
}

var entryRegex = regexp.MustCompile(`entry\.\d+`)

// GoogleFormsSender encaminha a avaliação anônima para um formulário Google Forms configurado.
type GoogleFormsSender struct {
	formURL        string
	entryRatingID  string
	entryCommentID string
	client         *http.Client
	mu             sync.RWMutex
}

// NewGoogleFormsSender cria um sender para o endpoint de resposta do Google Forms.
// Converte automaticamente URLs do tipo /viewform para /formResponse e extrai IDs se presentes.
func NewGoogleFormsSender(formURL, entryRatingID, entryCommentID string, client *http.Client) *GoogleFormsSender {
	if client == nil {
		client = &http.Client{Timeout: 4 * time.Second}
	}

	cleanURL := strings.TrimSpace(formURL)
	ratingID := strings.TrimSpace(entryRatingID)
	commentID := strings.TrimSpace(entryCommentID)

	if parsed, err := url.Parse(cleanURL); err == nil && parsed.Scheme != "" {
		// Se o usuário passou parâmetros de consulta com entry.XXXX
		q := parsed.Query()
		if ratingID == "" {
			for key := range q {
				if strings.HasPrefix(key, "entry.") {
					ratingID = key
					break
				}
			}
		}
		// Normaliza /viewform para /formResponse
		parsed.Path = strings.Replace(parsed.Path, "/viewform", "/formResponse", 1)
		parsed.RawQuery = ""
		cleanURL = parsed.String()
	}

	return &GoogleFormsSender{
		formURL:        cleanURL,
		entryRatingID:  ratingID,
		entryCommentID: commentID,
		client:         client,
	}
}

// discoverEntryIDs tenta descobrir os IDs dos campos entry.XXXX automaticamente se não informados.
func (g *GoogleFormsSender) discoverEntryIDs(ctx context.Context) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.entryRatingID != "" || g.formURL == "" {
		return
	}

	viewURL := strings.Replace(g.formURL, "/formResponse", "/viewform", 1)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, viewURL, http.NoBody)
	if err != nil {
		return
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return
	}

	matches := entryRegex.FindAllString(string(body), -1)
	seen := make(map[string]bool)
	var entries []string
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			entries = append(entries, m)
		}
	}

	if len(entries) > 0 {
		g.entryRatingID = entries[0]
	}
	if len(entries) > 1 && g.entryCommentID == "" {
		g.entryCommentID = entries[1]
	}
}

// Send realiza uma requisição POST com a nota fechada e o comentário opcional para o Google Forms.
func (g *GoogleFormsSender) Send(ctx context.Context, fb Feedback) error {
	if g.formURL == "" {
		return nil
	}

	g.mu.RLock()
	ratingID := g.entryRatingID
	commentID := g.entryCommentID
	g.mu.RUnlock()

	if ratingID == "" && commentID == "" {
		g.discoverEntryIDs(ctx)
		g.mu.RLock()
		ratingID = g.entryRatingID
		commentID = g.entryCommentID
		g.mu.RUnlock()
	}

	if ratingID == "" && commentID == "" {
		return nil
	}

	form := url.Values{}
	ratingText := fb.Rating.Description()

	// Caso tenha 2 campos configurados separadamente:
	if ratingID != "" && commentID != "" {
		form.Set(ratingID, ratingText)
		if fb.Comment != "" {
			form.Set(commentID, fb.Comment)
		}
	} else {
		// Caso tenha 1 campo: envia o ícone, seu significado e o comentário descritivo juntos
		targetID := ratingID
		if targetID == "" {
			targetID = commentID
		}
		content := ratingText
		if fb.Comment != "" {
			content = fmt.Sprintf("%s — %s", ratingText, fb.Comment)
		}
		form.Set(targetID, content)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.formURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("falha ao criar requisição para Google Forms: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("falha ao submeter formulário externo: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("google forms respondeu com status %d", resp.StatusCode)
	}

	return nil
}
