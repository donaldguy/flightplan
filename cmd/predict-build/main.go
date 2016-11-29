package main

import (
	"fmt"
	"os"

	"github.com/concourse/fly/rc"
	"github.com/donaldguy/flightplan"
	flags "github.com/jessevdk/go-flags"
	"github.com/k0kubun/pp"
)

type options struct {
	Target       rc.TargetName `short:"t" long:"target" description:"Fly target to monitor" required:"true"`
	PipelineName string        `short:"p" long:"pipeline" description:"Which pipeline to examine" required:"true"`
}

func dieIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	var opts options

	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	parser.Usage = "-t [TARGET] -p [PIPELINE] <resource>"

	args, err := parser.Parse()
	dieIf(err)

	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Must specify a resource")
		os.Exit(1)
	}

	target, err := rc.LoadTarget(opts.Target)
	dieIf(err)
	pipelineConfig, _, _, _, err := target.Team().PipelineConfig(opts.PipelineName)
	dieIf(err)
	pipeline := flightplan.NewPipeline(opts.PipelineName, &pipelineConfig)

	pp.Print(pipeline.GraphStartingFrom(args[0]))
}
