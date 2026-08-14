package ecs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"

	"github.com/takaishi/aws-tui/internal/ui"
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

	client := NewClient(cfg)
	var root *ui.Screen
	if app.CLI.Cluster != "" {
		title := fmt.Sprintf("Amazon ECS (%s) > %s", cfg.Region, app.CLI.Cluster)
		root = servicesScreen(client, title, app.CLI.Cluster)
	} else {
		root = clustersScreen(client, cfg.Region)
	}
	return ui.Run(ctx, root, ui.WithColumns())
}
