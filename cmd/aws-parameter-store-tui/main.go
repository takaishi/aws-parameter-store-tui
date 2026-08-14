package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	awsparameterstoretui "github.com/takaishi/aws-parameter-store-tui"
)

var Version = "dev"
var Revision = "HEAD"

func main() {
	awsparameterstoretui.Version = Version
	awsparameterstoretui.Revision = Revision

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	if err := awsparameterstoretui.RunCLI(ctx, os.Args[1:]); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
