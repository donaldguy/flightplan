package main

import (
	"fmt"
	"os"

	"github.com/concourse/fly/rc"
	"github.com/donaldguy/flightplan"
	"github.com/k0kubun/pp"

	flags "github.com/jessevdk/go-flags"
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

	pipeline, err := flightplan.NewPipeline(target.Team(), opts.PipelineName)
	dieIf(err)

	graph := pipeline.GraphStartingFrom(args[0])
	//rn := graph.Start

	pp.Print(graph.Start)
	printResource(graph.Start, "")
}

func printResource(r *flightplan.ResourceNode, buffer string) {
	fmt.Printf("%sResource: %s", buffer, r.Name)
	if len(r.Passed) > 0 {
		fmt.Printf(" (passed: %v)\n", r.Passed)
	} else {
		fmt.Println("")
	}

	for _, tj := range r.TriggeredJobs {
		printJob(tj, "  "+buffer)
	}
}

func printJob(j *flightplan.JobNode, buffer string) {
	fmt.Printf("%sJob %s\n", buffer, j.Name)
	for _, o := range j.Outputs {
		printResource(o, "  "+buffer)
	}
}
