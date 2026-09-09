// Package main inicializa a aplicação desktop Falaêh com Wails v2.
package main

import (
	"embed"
	"io/fs"
	"log/slog"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/themarcosramos/falaeh/backend/internal/exercise"
	"github.com/themarcosramos/falaeh/backend/internal/feedback"
	"github.com/themarcosramos/falaeh/backend/internal/game"
	"github.com/themarcosramos/falaeh/backend/internal/gamification"
	"github.com/themarcosramos/falaeh/backend/internal/httpapi"
)

//go:embed all:frontend
var frontendAssets embed.FS

//go:embed data/*.json
var embeddedData embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Subdiretório do frontend embutido
	assetsFS, err := fs.Sub(frontendAssets, "frontend")
	if err != nil {
		logger.Error("falha ao carregar assets do frontend", slog.Any("error", err))
		assetsFS = frontendAssets
	}

	// Subdiretório dos dados de exercícios embutidos
	dataFS, err := fs.Sub(embeddedData, "data")
	if err != nil {
		logger.Error("falha ao carregar dados de exercícios embutidos", slog.Any("error", err))
		dataFS = embeddedData
	}

	exerciseRepo, err := exercise.NewJSONRepository(dataFS)
	if err != nil {
		logger.Error("falha ao carregar repositório de exercícios", slog.Any("error", err))
		os.Exit(1)
	}

	exerciseService := exercise.NewService(exerciseRepo)
	gameManager := game.NewManager(exerciseService, gamification.DefaultRules(), game.Config{})

	var feedbackSender feedback.Sender
	if formURL := os.Getenv("FEEDBACK_GOOGLE_FORMS_URL"); formURL != "" {
		entryRatingID := os.Getenv("FEEDBACK_GOOGLE_FORMS_ENTRY_ID")
		entryCommentID := os.Getenv("FEEDBACK_GOOGLE_FORMS_COMMENT_ENTRY_ID")
		feedbackSender = feedback.NewGoogleFormsSender(formURL, entryRatingID, entryCommentID, nil)
	}
	feedbackService := feedback.NewService(feedbackSender, logger)

	apiHandler := httpapi.NewRouter(logger, exerciseService, gameManager, feedbackService)
	app := NewApp()

	err = wails.Run(&options.App{
		Title:     "Falaêh",
		Width:     1100,
		Height:    760,
		MinWidth:  800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets:  assetsFS,
			Handler: apiHandler,
		},
		BackgroundColour: &options.RGBA{R: 243, G: 244, B: 246, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			BackdropType:         windows.Mica,
		},
		Mac: &mac.Options{
			TitleBar:             mac.TitleBarDefault(),
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			Appearance:           mac.DefaultAppearance,
		},
		Linux: &linux.Options{
			Icon:                appIcon,
			WindowIsTranslucent: false,
		},
	})

	if err != nil {
		logger.Error("erro fatal ao executar wails", slog.Any("error", err))
		os.Exit(1)
	}
}
