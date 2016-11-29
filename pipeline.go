package flightplan

import (
	"fmt"

	"github.com/concourse/go-concourse/concourse"
)

type Pipeline struct {
	Name             string
	*Repos           //git.go
	*ResourcesByType //resources.go
}

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
		Repos:           repos(&config),
		ResourcesByType: resourcesByType(&config),
	}, nil
}
