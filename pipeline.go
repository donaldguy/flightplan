package flightplan

import "github.com/concourse/atc"

type ResourceName string
type Paths []string

type ResourcePaths map[ResourceName]Paths

type Repo struct {
	URI RepoURI
	ResourcePaths
}

type RepoURI string
type Repos map[RepoURI]Repo

type Pipeline struct {
	Name  string
	Repos Repos
}

func NewPipeline(name string, config *atc.Config) (p Pipeline) {
	p.Name = name
	p.fillRepos(config)

	return
}

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
