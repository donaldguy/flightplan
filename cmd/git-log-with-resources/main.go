package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	git "gopkg.in/libgit2/git2go.v24"

	"github.com/concourse/fly/rc"
	"github.com/donaldguy/flightplan"
	"github.com/fatih/color"
	flags "github.com/jessevdk/go-flags"
)

type Options struct {
	Target          rc.TargetName `short:"t" long:"target" description:"Fly target to monitor" required:"true"`
	PipelineName    string        `short:"p" long:"pipeline" description:"Which pipeline to examine" required:"true"`
	NumberOfCommits uint          `short:"n" long:"commits" description:"Number of commits to show in output" default:"10"`
}

func dieIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	var opts Options

	parser := flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	parser.Usage = "-t [TARGET] -p [PIPELINE] <git repo path>"
	parser.NamespaceDelimiter = "-"

	args, err := parser.Parse()
	dieIf(err)

	target, err := rc.LoadTarget(opts.Target)
	dieIf(err)
	pipelineConfig, _, _, _, err := target.Team().PipelineConfig(opts.PipelineName)
	dieIf(err)
	pipeline := flightplan.NewPipeline(opts.PipelineName, &pipelineConfig)

	repo, err := git.OpenRepository(args[0])
	dieIf(err)
	head, err := repo.Head()
	dieIf(err)

	headObj, err := repo.Lookup(head.Target())
	dieIf(err)
	headGCommit, err := headObj.AsCommit()
	dieIf(err)

	headCommit := &flightplan.GitCommit{Repo: repo, Commit: headGCommit}

	var i uint
	for i = 0; i < opts.NumberOfCommits; i++ {
		fmt.Print(color.YellowString("%s", headCommit.Id().String()[0:7]))
		fmt.Print(" ")
		fmt.Print(strings.Split(headCommit.Message(), "\n")[0])

		triggeredResources, err := headCommit.ResourcesTriggeredIn(&pipeline)
		dieIf(err)

		//drop common prefixes
		commonPrefixes := regexp.MustCompile(`^(?:git-)|(?:src-)`)
		for i, v := range triggeredResources {
			triggeredResources[i] = flightplan.ResourceName(commonPrefixes.ReplaceAllString(string(v), ""))
		}

		color.Cyan(" %v", triggeredResources)
		headCommit = headCommit.Parent(0)
		dieIf(err)
	}
}
