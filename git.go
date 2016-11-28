package flightplan

import (
	"fmt"
	"strings"

	"github.com/concourse/atc"

	git "gopkg.in/libgit2/git2go.v24"
)

type Repos map[RepoURI]Repo
type RepoURI string
type Repo struct {
	URI RepoURI
	ResourcePaths
}

type ResourcePaths map[ResourceName]Paths
type Paths []string

func repos(config *atc.Config) *Repos {
	repos := make(Repos)
	for _, resource := range config.Resources {
		if resource.Type != "git" {
			continue
		}

		uri := RepoURI(resource.Source["uri"].(string))

		if _, ok := repos[uri]; !ok {
			repo := Repo{URI: uri}
			repo.ResourcePaths = make(ResourcePaths)

			repos[uri] = repo
		}

		if resource.Source["paths"] != nil {
			resourceName := ResourceName(resource.Name)
			paths := resource.Source["paths"].([]interface{})
			repos[uri].ResourcePaths[resourceName] = make(Paths, len(paths))

			for i, v := range paths {
				repos[uri].ResourcePaths[resourceName][i] = v.(string)
			}
		} else {
			repos[uri].ResourcePaths[ResourceName(resource.Name)] = make(Paths, 0)
		}

		// TODO: deal with ignore_paths too

	}
	return &repos
}

type GitCommit struct {
	// keep an explicit reference to the repo, cause commit.Owner() frequently returns pointer to free'd memory -_-
	Repo *git.Repository
	*git.Commit
}

func (c *GitCommit) Parent(n uint) *GitCommit {
	return &GitCommit{
		Repo:   c.Repo,
		Commit: c.Commit.Parent(n),
	}
}

type GitRange struct {
	Old *GitCommit
	New *GitCommit
}

func (c *GitCommit) ResourcesTriggeredIn(pipeline *Pipeline) (resources []ResourceName, err error) {
	if strings.Contains(c.Message(), "[skip ci]") || strings.Contains(c.Message(), "[ci skip]") {
		return []ResourceName{}, nil
	}

	return (&GitRange{
		Old: c.Parent(0),
		New: c,
	}).ResourcesTriggeredIn(pipeline)
}

func (r *GitRange) ResourcesTriggeredIn(pipeline *Pipeline) (resources []ResourceName, err error) {
	resources = make([]ResourceName, 0)

	if r.Old.Repo != r.New.Repo {
		return resources, fmt.Errorf("Mismatched repos in GitRange!")
	}
	repo := r.Old.Repo

	origin, err := repo.Remotes.Lookup("origin")
	if err != nil {
		return
	}
	uri := RepoURI(origin.Url())
	pathsCollection, ok := (*pipeline.Repos)[uri]
	if !ok {
		return resources, fmt.Errorf("No resources in pipeline reference uri: %s", uri)
	}

	oldTree, err := r.Old.Tree()
	if err != nil {
		return
	}
	newTree, err := r.New.Tree()
	if err != nil {
		return
	}

	for resourceName, paths := range pathsCollection.ResourcePaths {
		opts := &git.DiffOptions{
			Pathspec: []string(paths),
		}

		var diff *git.Diff
		diff, err = repo.DiffTreeToTree(oldTree, newTree, opts)
		if err != nil {
			return
		}

		var diffStats *git.DiffStats
		diffStats, err = diff.Stats()
		if err != nil {
			return
		}

		if diffStats.FilesChanged() != 0 {
			resources = append(resources, resourceName)
		}
	}

	return resources, nil
}
