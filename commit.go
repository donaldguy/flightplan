package flightplan

import (
	"fmt"
	"strings"

	git "gopkg.in/libgit2/git2go.v24"
)

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

func (c *GitCommit) ResourcesTriggered(pipeline *Pipeline) (resources []ResourceName, err error) {
	resources = make([]ResourceName, 0)

	if strings.Contains(c.Message(), "[skip ci]") || strings.Contains(c.Message(), "[ci skip]") {
		return
	}

	repo := c.Repo

	origin, err := repo.Remotes.Lookup("origin")
	if err != nil {
		return
	}
	uri := RepoURI(origin.Url())
	pathsCollection, ok := pipeline.Repos[uri]
	if !ok {
		return resources, fmt.Errorf("No resources in pipeline reference uri: %s", uri)
	}

	thisTree, err := c.Tree()
	if err != nil {
		return
	}
	parentTree, err := c.Parent(0).Tree()
	if err != nil {
		return
	}

	for resourceName, paths := range pathsCollection.ResourcePaths {
		opts := &git.DiffOptions{
			Pathspec: []string(paths),
		}

		var diff *git.Diff
		diff, err = repo.DiffTreeToTree(parentTree, thisTree, opts)
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
