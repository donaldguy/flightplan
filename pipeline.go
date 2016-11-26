package flightplan

import (
	"github.com/concourse/atc"
	configParser "github.com/concourse/atc/config"
)

type Pipeline struct {
	Name string
	Repos
	Graph
}

func NewPipeline(name string, config *atc.Config) (p Pipeline) {
	p.Name = name
	p.fillRepos(config)
	p.classifyInputs(config)
	p.classifyOutputs(config)

	return
}

type Repos map[RepoURI]Repo
type RepoURI string

type Repo struct {
	URI RepoURI
	ResourcePaths
}

type ResourcePaths map[ResourceName]Paths
type ResourceName string
type Paths []string

func (p *Pipeline) fillRepos(config *atc.Config) {
	p.Repos = make(Repos)
	for _, resource := range config.Resources {
		if resource.Type != "git" {
			continue
		}

		uri := RepoURI(resource.Source["uri"].(string))

		if _, ok := p.Repos[uri]; !ok {
			repo := Repo{URI: uri}
			repo.ResourcePaths = make(ResourcePaths)

			p.Repos[uri] = repo
		}

		if resource.Source["paths"] != nil {
			resourceName := ResourceName(resource.Name)
			paths := resource.Source["paths"].([]interface{})
			p.Repos[uri].ResourcePaths[resourceName] = make(Paths, len(paths))

			for i, v := range paths {
				p.Repos[uri].ResourcePaths[resourceName][i] = v.(string)
			}
		} else {
			p.Repos[uri].ResourcePaths[ResourceName(resource.Name)] = make(Paths, 0)
		}

		// TODO: deal with ignore_paths too

	}
	return
}

type Graph struct {
	Entrypoints           map[ResourceName][]Entrypoint
	MidtriggersByJob      map[JobName][]*Midtrigger
	MidtriggersByResource map[ResourceName][]*Midtrigger
	Byproducts            map[JobName][]ResourceName
	Products              map[ResourceName]JobName
}

type JobName string

type Entrypoint struct {
	ResourceName
	TriggeredJob JobName
}

type Midtrigger struct {
	ResourceName
	Passed       []JobName
	TriggeredJob JobName
}

// classify inputs as either:
//  entrypoints:             input -> job
//  midtriggers: [passed] -> input -> job
func (p *Pipeline) classifyInputs(config *atc.Config) {
	p.Graph.Entrypoints = make(map[ResourceName][]Entrypoint)
	p.Graph.MidtriggersByJob = make(map[JobName][]*Midtrigger)
	p.Graph.MidtriggersByResource = make(map[ResourceName][]*Midtrigger)

	for _, job := range config.Jobs {
		jobName := JobName(job.Name)
		for _, input := range configParser.JobInputs(job) {

			if input.Trigger {
				inputName := ResourceName(input.Resource)

				// is an entrypoint, record as such
				if len(input.Passed) == 0 {
					if _, ok := p.Graph.Entrypoints[inputName]; !ok {
						p.Graph.Entrypoints[inputName] = []Entrypoint{}
					}
					p.Graph.Entrypoints[inputName] = append(
						p.Graph.Entrypoints[inputName],
						Entrypoint{inputName, jobName},
					)
				} else {
					// it is a middle stage trigger

					mt := &Midtrigger{
						ResourceName: inputName,
						Passed:       make([]JobName, len(input.Passed)),
						TriggeredJob: jobName,
					}
					for i, name := range input.Passed {
						mt.Passed[i] = JobName(name)
					}

					if _, ok := p.Graph.MidtriggersByJob[jobName]; !ok {
						p.Graph.MidtriggersByJob[jobName] = []*Midtrigger{}
					}
					p.Graph.MidtriggersByJob[jobName] = append(
						p.Graph.MidtriggersByJob[jobName],
						mt,
					)

					if _, ok := p.Graph.MidtriggersByResource[inputName]; !ok {
						p.Graph.MidtriggersByResource[inputName] = []*Midtrigger{}
					}
					p.Graph.MidtriggersByResource[inputName] = append(
						p.Graph.MidtriggersByResource[inputName],
						mt,
					)
				}
			}
		}
	}
}

func (p *Pipeline) classifyOutputs(config *atc.Config) {
	p.Graph.Products = make(map[ResourceName]JobName)
	p.Graph.Byproducts = make(map[JobName][]ResourceName)

	for _, job := range config.Jobs {
		jobName := JobName(job.Name)
		for _, output := range configParser.JobOutputs(job) {
			outputName := ResourceName(output.Resource)

			if _, isMiddtrigger := p.Graph.MidtriggersByResource[outputName]; !isMiddtrigger {
				p.Graph.Products[outputName] = jobName
			}

			if _, ok := p.Graph.Byproducts[jobName]; !ok {
				p.Graph.Byproducts[jobName] = []ResourceName{}
			}
			p.Graph.Byproducts[jobName] = append(p.Graph.Byproducts[jobName], outputName)
		}
	}
}
