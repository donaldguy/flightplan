package flightplan

import (
	"fmt"

	"github.com/concourse/go-concourse/concourse"
)

// Pipeline is a parsed representation of a concourse pipeline config suitable for generating
// graphs and using to resolve triggered resources from commits
type Pipeline struct {
	Name            string
	repos           *repos           //git.go
	resourcesByType *resourcesByType //resources.go
}

// NewPipeline contacts the concourse api endpoint corresponding to `team`, fetches
// the PipelineConfig for pipeline with `name`, then parses and returns a `flightplan.Pipeline`
func NewPipeline(team concourse.Team, name string) (*Pipeline, error) {
	config, _, _, exists, err := team.PipelineConfig(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Team %s has no pipeline named %s", team.Name(), name)
	}

	return &Pipeline{
		Name:            name,
		repos:           newRepos(&config),
		resourcesByType: newResourcesByType(&config),
	}, nil
}
