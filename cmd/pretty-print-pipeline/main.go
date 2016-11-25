package main

import (
	"fmt"
	"os"

	"github.com/concourse/fly/rc"
	"github.com/donaldguy/flightplan"
	flags "github.com/jessevdk/go-flags"
	"github.com/k0kubun/pp"
)

type Options struct {
	Target rc.TargetName `short:"t" long:"target" description:"Fly target to monitor" env:"FLIGHT_TRACKER_SERVER"`
}

func main() {
	var opts Options

	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	parser.NamespaceDelimiter = "-"

	args, err := parser.Parse()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	target, err := rc.LoadTarget(opts.Target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	apiClient := target.Client()
	p, _, _, _, err := apiClient.Team("main").PipelineConfig(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	p2 := flightplan.NewPipeline(args[0], &p)
	pp.Print(p2)
}
