package main

import (
	"fmt"
	"os"

	"github.com/concourse/fly/rc"
	"github.com/donaldguy/flightplan"
	flags "github.com/jessevdk/go-flags"
)

type Options struct {
	Target       rc.TargetName `short:"t" long:"target" description:"Fly target to monitor" required:"true"`
	PipelineName string        `short:"p" long:"pipeline" description:"Which pipeline to examine" required:"true"`
}

func dieIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var UsedResources map[flightplan.ResourceName]bool

func main() {
	var opts Options

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

	UsedResources = make(map[flightplan.ResourceName]bool, 0)
	doResource(&pipeline.Graph, flightplan.ResourceName(args[0]))
}

func doResource(g *flightplan.Graph, r flightplan.ResourceName) {
	UsedResources[r] = true
	fmt.Printf("Triggered Resource: %s\n", r)
	if _, done := g.Products[r]; done {
		return
	}
	for _, entrypoint := range g.Entrypoints[r] {
		doJob(g, entrypoint.TriggeredJob)
	}

	for _, midtriggers := range g.MidtriggersByResource[r] {
		fmt.Printf("\nif %v pass(es):\n", midtriggers.Passed)
		doJob(g, midtriggers.TriggeredJob)
	}
}

func doJob(g *flightplan.Graph, j flightplan.JobName) {
	fmt.Printf("Triggered Job: %s\n", j)
	for resource, entrypoints := range g.Entrypoints {
		for _, entrypoint := range entrypoints {
			if entrypoint.TriggeredJob == j {
				if _, alreadyUsed := UsedResources[resource]; !alreadyUsed {
					fmt.Printf("(Also need %s)\n", resource)
				}
			}
		}
	}
	for _, r := range g.Byproducts[j] {
		doResource(g, r)
	}
}
