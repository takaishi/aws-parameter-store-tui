package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	pstui "github.com/takaishi/aws-parameter-store-tui"
)

var Version = "dev"
var Revision = "HEAD"

func main() {
	pstui.Version = Version
	pstui.Revision = Revision

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	if err := pstui.RunCLI(ctx, os.Args[1:]); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
