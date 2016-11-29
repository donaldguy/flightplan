package flightplan

import (
	"fmt"
	"strings"

	"github.com/concourse/atc"

	git "gopkg.in/libgit2/git2go.v24"
)

type repos map[repoURI]repo
type repoURI string
type repo struct {
	URI repoURI
	resourcePaths
}

type resourcePaths map[ResourceName]paths
type paths []string

func newRepos(config *atc.Config) *repos {
	rs := make(repos)
	for _, resource := range config.Resources {
		if resource.Type != "git" {
			continue
		}

		uri := repoURI(resource.Source["uri"].(string))

		if _, ok := rs[uri]; !ok {
			r := repo{URI: uri}
			r.resourcePaths = make(resourcePaths)

			rs[uri] = r
		}

		if resource.Source["paths"] != nil {
			resourceName := ResourceName(resource.Name)
			ps := resource.Source["paths"].([]interface{})
			rs[uri].resourcePaths[resourceName] = make(paths, len(ps))

			for i, p := range ps {
				rs[uri].resourcePaths[resourceName][i] = p.(string)
			}
		} else {
			rs[uri].resourcePaths[ResourceName(resource.Name)] = make(paths, 0)
		}

		// TODO: deal with ignore_paths too

	}
	return &rs
}

// GitCommit represents a git commit; it exposes all the functionality of git2go.Commit
// but also adds a function for figuring out which resources in a pipeline are triggered
type GitCommit struct {
	// keep an explicit reference to the repo, cause commit.Owner() frequently returns pointer to free'd memory -_-
	Repo *git.Repository
	*git.Commit
}

// Parent is like git2go.Commit.Parent but returns a GitCommit
func (c *GitCommit) Parent(n uint) *GitCommit {
	return &GitCommit{
		Repo:   c.Repo,
		Commit: c.Commit.Parent(n),
	}
}

// GitRange represents a range of GitCommits, which make up the endpoints of a git diff
type GitRange struct {
	Old *GitCommit
	New *GitCommit
}

// ResourcesTriggeredIn returns the concourse resources in `pipeline`  whose whitelists saw changes in GitCommit `c`
func (c *GitCommit) ResourcesTriggeredIn(pipeline *Pipeline) (resources []string, err error) {
	if strings.Contains(c.Message(), "[skip ci]") || strings.Contains(c.Message(), "[ci skip]") {
		return []string{}, nil
	}

	return (&GitRange{
		Old: c.Parent(0),
		New: c,
	}).ResourcesTriggeredIn(pipeline)
}

// ResourcesTriggeredIn returns the concourse resources in `pipeline`  whose whitelists saw changes in GitRange `r`
func (r *GitRange) ResourcesTriggeredIn(pipeline *Pipeline) (resources []string, err error) {
	resources = make([]string, 0)

	if r.Old.Repo != r.New.Repo {
		return resources, fmt.Errorf("Mismatched repos in GitRange!")
	}
	repo := r.Old.Repo

	origin, err := repo.Remotes.Lookup("origin")
	if err != nil {
		return
	}
	uri := repoURI(origin.Url())
	pathsCollection, ok := (*pipeline.repos)[uri]
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

	for resourceName, paths := range pathsCollection.resourcePaths {
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
			resources = append(resources, string(resourceName))
		}
	}

	return resources, nil
}
