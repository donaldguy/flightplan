package flightplan

import "github.com/concourse/atc"

type Pipeline struct {
	Name             string
	*Repos           //git.go
	*ResourcesByType //resources.go
}

func NewPipeline(name string, config *atc.Config) (p Pipeline) {
	p.Name = name
	p.Repos = repos(config)

	//resources.go
	p.ResourcesByType = resourcesByType(config)

	return
}
