package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	sstui "github.com/takaishi/aws-ss-tui"
)

var Version = "dev"
var Revision = "HEAD"

func main() {
	sstui.Version = Version
	sstui.Revision = Revision

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	if err := sstui.RunCLI(ctx, os.Args[1:]); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
