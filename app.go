package sstui

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	tea "github.com/charmbracelet/bubbletea"
)

type App struct {
	CLI *CLI
}

func New(cli *CLI) *App {
	return &App{
		CLI: cli,
	}
}

func (app *App) Run(ctx context.Context) error {
	var opts []func(*config.LoadOptions) error
	if app.CLI.Profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(app.CLI.Profile))
	}
	if app.CLI.Region != "" {
		opts = append(opts, config.WithRegion(app.CLI.Region))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	m := newModel(ctx, NewSSMClient(cfg), cfg.Region)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}
	return nil
}
