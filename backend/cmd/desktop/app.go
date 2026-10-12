package main

import (
	"context"
)

// App gerencia o ciclo de vida da aplicação desktop Falaêh.
type App struct {
	ctx context.Context
}

// NewApp cria uma nova instância da aplicação desktop.
func NewApp() *App {
	return &App{}
}

// startup é chamado pelo Wails quando a janela desktop é inicializada.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown é chamado pelo Wails quando a aplicação desktop é encerrada.
func (a *App) shutdown(_ context.Context) {
}
