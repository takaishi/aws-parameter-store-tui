package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/takaishi/aws-tui/internal/ec2"
)

var Version = "dev"
var Revision = "HEAD"

func main() {
	ec2.Version = Version
	ec2.Revision = Revision

	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()
	if err := ec2.RunCLI(ctx, os.Args[1:]); err != nil {
		log.Printf("error: %v", err)
		os.Exit(1)
	}
}
