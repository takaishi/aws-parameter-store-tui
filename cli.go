package pstui

import (
	"context"
	"fmt"

	"github.com/alecthomas/kong"
)

var Version = "dev"
var Revision = "HEAD"

type CLI struct {
	Profile string      `name:"profile" help:"AWS profile to use" env:"AWS_PROFILE"`
	Region  string      `name:"region" help:"AWS region to use" env:"AWS_REGION"`
	Version VersionFlag `name:"version" help:"show version"`
}

type VersionFlag string

func (v VersionFlag) Decode(ctx *kong.DecodeContext) error { return nil }
func (v VersionFlag) IsBool() bool                         { return true }
func (v VersionFlag) BeforeApply(app *kong.Kong, vars kong.Vars) error {
	fmt.Printf("%s-%s\n", Version, Revision)
	app.Exit(0)
	return nil
}

func RunCLI(ctx context.Context, args []string) error {
	cli := CLI{}
	parser, err := kong.New(&cli,
		kong.Name("pstui"),
		kong.Description("TUI for browsing AWS Systems Manager Parameter Store"),
	)
	if err != nil {
		return fmt.Errorf("error creating CLI parser: %w", err)
	}
	_, err = parser.Parse(args)
	if err != nil {
		return fmt.Errorf("error parsing CLI: %w", err)
	}
	app := New(&cli)
	return app.Run(ctx)
}
