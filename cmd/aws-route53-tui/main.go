package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/takaishi/aws-tui/internal/route53"
)

var Version = "dev"
var Revision = "HEAD"

func main() {
	route53.Version = Version
	route53.Revision = Revision

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	if err := route53.RunCLI(ctx, os.Args[1:]); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
