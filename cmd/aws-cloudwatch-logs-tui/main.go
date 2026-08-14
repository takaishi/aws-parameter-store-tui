package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/takaishi/aws-tui/internal/cloudwatchlogs"
)

var Version = "dev"
var Revision = "HEAD"

func main() {
	cloudwatchlogs.Version = Version
	cloudwatchlogs.Revision = Revision

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	if err := cloudwatchlogs.RunCLI(ctx, os.Args[1:]); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
