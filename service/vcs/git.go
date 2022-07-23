package vcs

import (
	"context"
	"github.com/beldeveloper/app-lego/model"
	appOs "github.com/beldeveloper/app-lego/service/os"
	"github.com/beldeveloper/go-errors-context"
	"log"
	"regexp"
	"strings"
)

const (
	// DefaultCfgFile defines the default name of the repository configuration file.
	DefaultCfgFile = "app-lego.yml"
)

// NewGit creates a new instance of the Git VCS service.
func NewGit(workDir model.FilePath, os appOs.Service) Service {
	return Git{
		workDir:        string(workDir + "/" + model.RepositoriesDir),
		os:             os,
		remoteBranchRx: regexp.MustCompile("^([a-f0-9]+)\\s+refs/(heads|tags)/(.*)$"),
	}
}

// Git implements the VCS service for Git.
type Git struct {
	workDir        string
	os             appOs.Service
	remoteBranchRx *regexp.Regexp
}

// DownloadRepository downloads the repository to the working directory.
func (g Git) DownloadRepository(ctx context.Context, r model.Repository) error {
	_, err := g.os.RunCmd(ctx, model.Cmd{
		Name: "git",
		Args: []string{"clone", r.Name, r.Alias},
		Dir:  g.workDir,
		Log:  true,
	})
	return errors.WrapContext(err, errors.Context{
		Path:   "service.vcs.git.DownloadRepository",
		Params: errors.Params{"repository": r.ID},
	})
}

// Branches parses the branches and tags from the remote repository.
func (g Git) Branches(ctx context.Context, r model.Repository) ([]model.Branch, error) {
	out, err := g.os.RunCmd(ctx, model.Cmd{
		Name: "git",
		Args: []string{"ls-remote"},
		Dir:  g.workDir + "/" + r.Alias,
	})
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "service.vcs.git.Branches: ls",
			Params: errors.Params{"repository": r.ID},
		})
	}
	rows := strings.Split(out, "\n")
	branches := make([]model.Branch, 0, len(rows))
	b := model.Branch{RepositoryID: r.ID}
	for _, r := range rows {
		r := strings.TrimSpace(r)
		matches := g.remoteBranchRx.FindStringSubmatch(r)
		if len(matches) < 4 {
			continue
		}
		b.Hash = matches[1]
		b.Name = matches[3]
		switch matches[2] {
		case "heads":
			b.Type = model.BranchTypeHead
		case "tags":
			b.Type = model.BranchTypeTag
		default:
			continue
		}
		branches = append(branches, b)
	}
	return branches, nil
}

// SwitchBranch checkouts to the specific branch and pulls the updates.
func (g Git) SwitchBranch(ctx context.Context, r model.Repository, b model.Branch) error {
	_, err := g.os.RunCmd(ctx, model.Cmd{
		Name: "git",
		Args: []string{"fetch"},
		Dir:  g.workDir + "/" + r.Alias,
		Log:  true,
	})
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "service.vcs.git.SwitchBranch: fetch",
			Params: errors.Params{"repository": r.ID},
		})
	}
	newBranch := b.Name
	if b.Type == model.BranchTypeTag {
		newBranch = "tags/" + b.Name
	}
	_, err = g.os.RunCmd(ctx, model.Cmd{
		Name: "git",
		Args: []string{"reset", "--hard"},
		Dir:  g.workDir + "/" + r.Alias,
		Log:  true,
	})
	if err != nil {
		log.Println(errors.WrapContext(err, errors.Context{
			Path:   "service.vcs.git.SwitchBranch: reset",
			Params: errors.Params{"repository": r.ID},
		}))
	}
	_, err = g.os.RunCmd(ctx, model.Cmd{
		Name: "git",
		Args: []string{"checkout", newBranch},
		Dir:  g.workDir + "/" + r.Alias,
		Log:  true,
	})
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "service.vcs.git.SwitchBranch: checkout",
			Params: errors.Params{"repository": r.ID, "branchName": b.Name},
		})
	}
	if b.Type == model.BranchTypeTag {
		return nil
	}
	_, err = g.os.RunCmd(ctx, model.Cmd{
		Name: "git",
		Args: []string{"pull"},
		Dir:  g.workDir + "/" + r.Alias,
		Log:  true,
	})
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "service.vcs.git.SwitchBranch: pull",
			Params: errors.Params{"repository": r.ID, "branch": b.ID},
		})
	}
	return nil
}
