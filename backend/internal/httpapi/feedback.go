package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/themarcosramos/falaeh/backend/internal/feedback"
)

// FeedbackService define o contrato necessário para receber a avaliação.
type FeedbackService interface {
	Submit(ctx context.Context, rawRating, rawComment string) error
}

// FeedbackRequest transporta a avaliação fechada e comentário opcional enviados pelo usuário.
type FeedbackRequest struct {
	Rating  string `json:"rating" example:"gostei_muito"`
	Comment string `json:"comment,omitempty" example:"Adorei a missão dos sons!"`
}

// FeedbackResponse confirma o recebimento da avaliação.
type FeedbackResponse struct {
	Status  string `json:"status" example:"ok"`
	Message string `json:"message" example:"Muito obrigado pela sua avaliação!"`
}

// handleFeedback godoc
//
//	@Summary		Envia uma avaliação anônima da experiência do Fala Eh
//	@Description	Recebe exclusivamente uma opção fechada de avaliação e comentário opcional sem vincular qualquer identificador ou dado pessoal.
//	@Tags			feedback
//	@Accept			json
//	@Produce		json
//	@Param			request	body		FeedbackRequest		true	"Opção de avaliação fechada e comentário opcional"
//	@Success		200		{object}	FeedbackResponse	"Avaliação registrada com sucesso"
//	@Failure		400		{object}	ErrorResponse		"Opção de avaliação inválida ou vazia"
//	@Router			/api/v1/feedback [post]
func handleFeedback(logger *slog.Logger, svc FeedbackService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if svc == nil {
			writeError(w, logger, http.StatusServiceUnavailable, "serviço de avaliação indisponível")
			return
		}

		var req FeedbackRequest
		if !decodeJSONBody(w, r, logger, &req) {
			return
		}

		if err := svc.Submit(r.Context(), req.Rating, req.Comment); err != nil {
			if errors.Is(err, feedback.ErrInvalidRating) || errors.Is(err, feedback.ErrEmptyRating) {
				writeError(w, logger, http.StatusBadRequest, err.Error())
				return
			}

			// Falhas em integrações externas degradam graciosamente: não bloqueiam nem punem o jogador
			logger.Warn("avaliação processada com aviso no encaminhamento externo", slog.Any("error", err))
		}

		writeJSON(w, logger, http.StatusOK, FeedbackResponse{
			Status:  "ok",
			Message: "Muito obrigado pela sua avaliação!",
		})
	}
}
