package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/takaishi/aws-tui/internal/ecs"
)

var Version = "dev"
var Revision = "HEAD"

func main() {
	ecs.Version = Version
	ecs.Revision = Revision

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	if err := ecs.RunCLI(ctx, os.Args[1:]); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
