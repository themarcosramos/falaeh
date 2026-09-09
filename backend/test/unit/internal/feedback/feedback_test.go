package feedback_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/themarcosramos/falaeh/backend/internal/feedback"
)

func TestParseRating(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      feedback.Rating
		expectErr error
	}{
		{
			name:      "gostei_muito em minúsculas",
			input:     "gostei_muito",
			want:      feedback.RatingLoved,
			expectErr: nil,
		},
		{
			name:      "gostei com espaços e maiúsculas",
			input:     "  GOSTEI  ",
			want:      feedback.RatingLiked,
			expectErr: nil,
		},
		{
			name:      "mais_ou_menos válido",
			input:     "mais_ou_menos",
			want:      feedback.RatingNeutral,
			expectErr: nil,
		},
		{
			name:      "nao_gostei válido",
			input:     "nao_gostei",
			want:      feedback.RatingDisliked,
			expectErr: nil,
		},
		{
			name:      "entrada vazia",
			input:     "   ",
			want:      "",
			expectErr: feedback.ErrEmptyRating,
		},
		{
			name:      "valor desconhecido",
			input:     "odiei",
			want:      "",
			expectErr: feedback.ErrInvalidRating,
		},
		{
			name:      "tentativa de texto livre",
			input:     "o jogo é legal mas podia ter mais fases",
			want:      "",
			expectErr: feedback.ErrInvalidRating,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := feedback.ParseRating(tt.input)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) {
					t.Fatalf("erro esperado %v, obtido %v", tt.expectErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("erro inesperado: %v", err)
			}

			if got != tt.want {
				t.Errorf("obtido %q, esperado %q", got, tt.want)
			}
			if !got.IsValid() {
				t.Errorf("rating %q deveria ser válido", got)
			}
		})
	}
}

func TestSanitizeComment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "texto vazio",
			input: "   ",
			want:  "",
		},
		{
			name:  "texto com espaços nas pontas",
			input: "   muito legal o jogo   ",
			want:  "muito legal o jogo",
		},
		{
			name:  "texto longo truncado em 200 caracteres",
			input: strings.Repeat("a", 250),
			want:  strings.Repeat("a", 200),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := feedback.SanitizeComment(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeComment() = %q, esperado %q", got, tt.want)
			}
			if len([]rune(got)) > feedback.MaxCommentLength {
				t.Errorf("comentário tem %d caracteres, máximo é %d", len([]rune(got)), feedback.MaxCommentLength)
			}
		})
	}
}

type mockSender struct {
	sentFeedback feedback.Feedback
	errToReturn  error
	calls        int
}

func (m *mockSender) Send(_ context.Context, fb feedback.Feedback) error {
	m.calls++
	m.sentFeedback = fb
	return m.errToReturn
}

func TestService_Submit(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("sucesso sem sender externo", func(t *testing.T) {
		svc := feedback.NewService(nil, logger)
		err := svc.Submit(context.Background(), "gostei_muito", "excelente jogo!")
		if err != nil {
			t.Fatalf("esperado nil, obtido %v", err)
		}
	})

	t.Run("sucesso com sender externo configurado e comentário", func(t *testing.T) {
		sender := &mockSender{}
		svc := feedback.NewService(sender, logger)

		err := svc.Submit(context.Background(), "gostei", "adorei os sons")
		if err != nil {
			t.Fatalf("esperado nil, obtido %v", err)
		}
		if sender.calls != 1 {
			t.Errorf("chamadas = %d, esperado 1", sender.calls)
		}
		if sender.sentFeedback.Rating != feedback.RatingLiked {
			t.Errorf("rating enviado = %q, esperado %q", sender.sentFeedback.Rating, feedback.RatingLiked)
		}
		if sender.sentFeedback.Comment != "adorei os sons" {
			t.Errorf("comment enviado = %q, esperado %q", sender.sentFeedback.Comment, "adorei os sons")
		}
	})

	t.Run("avaliação inválida rejeitada antes do sender", func(t *testing.T) {
		sender := &mockSender{}
		svc := feedback.NewService(sender, logger)

		err := svc.Submit(context.Background(), "invalido", "")
		if !errors.Is(err, feedback.ErrInvalidRating) {
			t.Fatalf("esperado ErrInvalidRating, obtido %v", err)
		}
		if sender.calls != 0 {
			t.Errorf("sender não deveria ser chamado")
		}
	})

	t.Run("erro no sender externo repassado pelo serviço", func(t *testing.T) {
		senderErr := errors.New("timeout na conexão externa")
		sender := &mockSender{errToReturn: senderErr}
		svc := feedback.NewService(sender, logger)

		err := svc.Submit(context.Background(), "mais_ou_menos", "")
		if !errors.Is(err, senderErr) {
			t.Fatalf("esperado %v, obtido %v", senderErr, err)
		}
	})
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGoogleFormsSender(t *testing.T) {
	t.Parallel()

	t.Run("envio com sucesso para formulário externo com nota e comentário separados", func(t *testing.T) {
		var receivedBody string
		client := &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodPost {
					t.Errorf("método esperado POST, obtido %s", req.Method)
				}
				if ct := req.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
					t.Errorf("content-type esperado application/x-www-form-urlencoded, obtido %s", ct)
				}
				b, _ := io.ReadAll(req.Body)
				receivedBody = string(b)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("ok")),
				}, nil
			}),
		}

		sender := feedback.NewGoogleFormsSender("https://forms.google.com/test", "entry.111111", "entry.222222", client)
		err := sender.Send(context.Background(), feedback.Feedback{
			Rating:  feedback.RatingLoved,
			Comment: "muito divertido!",
		})
		if err != nil {
			t.Fatalf("esperado nil, obtido %v", err)
		}

		if !strings.Contains(receivedBody, "entry.111111=%F0%9F%98%84+Gostei+muito") {
			t.Errorf("rating com descrição esperado não encontrado no corpo: %s", receivedBody)
		}
		if !strings.Contains(receivedBody, "entry.222222=muito+divertido%21") {
			t.Errorf("comentário esperado não encontrado no corpo: %s", receivedBody)
		}
	})

	t.Run("envio unificado em 1 único campo com ícone significado e comentário juntos", func(t *testing.T) {
		var receivedBody string
		client := &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				b, _ := io.ReadAll(req.Body)
				receivedBody = string(b)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("ok")),
				}, nil
			}),
		}

		sender := feedback.NewGoogleFormsSender("https://forms.google.com/test", "entry.111111", "", client)
		err := sender.Send(context.Background(), feedback.Feedback{
			Rating:  feedback.RatingLiked,
			Comment: "adoramos o jogo",
		})
		if err != nil {
			t.Fatalf("esperado nil, obtido %v", err)
		}

		// Deve conter "🙂 Gostei — adoramos o jogo" encodado
		if !strings.Contains(receivedBody, "entry.111111=%F0%9F%99%82+Gostei+%E2%80%94+adoramos+o+jogo") {
			t.Errorf("campo único formatado não encontrado no corpo: %s", receivedBody)
		}
	})

	t.Run("descoberta automática de entry IDs via HTML da página do formulário", func(t *testing.T) {
		var postedBody string
		client := &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method == http.MethodGet {
					fakeHTML := `<html><body><input type="text" name="entry.987654"><textarea name="entry.123456"></textarea></body></html>`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(fakeHTML)),
					}, nil
				}
				b, _ := io.ReadAll(req.Body)
				postedBody = string(b)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("ok")),
				}, nil
			}),
		}

		sender := feedback.NewGoogleFormsSender("https://docs.google.com/forms/d/e/XYZ/viewform", "", "", client)
		err := sender.Send(context.Background(), feedback.Feedback{
			Rating:  feedback.RatingNeutral,
			Comment: "legal",
		})
		if err != nil {
			t.Fatalf("esperado nil, obtido %v", err)
		}

		if !strings.Contains(postedBody, "entry.987654=%F0%9F%98%90+Foi+mais+ou+menos") {
			t.Errorf("entry descoberto não encontrado no corpo do POST: %s", postedBody)
		}
		if !strings.Contains(postedBody, "entry.123456=legal") {
			t.Errorf("entry de comentário descoberto não encontrado no corpo: %s", postedBody)
		}
	})

	t.Run("configuração vazia degrada sem erro", func(t *testing.T) {
		sender := feedback.NewGoogleFormsSender("", "", "", nil)
		err := sender.Send(context.Background(), feedback.Feedback{Rating: feedback.RatingLoved})
		if err != nil {
			t.Fatalf("esperado nil, obtido %v", err)
		}
	})

	t.Run("servidor remoto retornando erro HTTP 500", func(t *testing.T) {
		client := &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("error")),
				}, nil
			}),
		}

		sender := feedback.NewGoogleFormsSender("https://forms.google.com/test", "entry.123456", "", client)
		err := sender.Send(context.Background(), feedback.Feedback{Rating: feedback.RatingDisliked})
		if err == nil {
			t.Fatal("esperava erro quando servidor responde com status 500")
		}
	})

	t.Run("contexto expirado cancela requisição", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		sender := feedback.NewGoogleFormsSender("https://forms.google.com/test", "entry.123456", "", nil)
		err := sender.Send(ctx, feedback.Feedback{Rating: feedback.RatingNeutral})
		if err == nil {
			t.Fatal("esperava erro por contexto cancelado")
		}
	})
}
