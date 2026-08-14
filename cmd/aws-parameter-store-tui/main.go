package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	apstui "github.com/takaishi/aws-parameter-store-tui"
)

var Version = "dev"
var Revision = "HEAD"

func main() {
	apstui.Version = Version
	apstui.Revision = Revision

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	if err := apstui.RunCLI(ctx, os.Args[1:]); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
