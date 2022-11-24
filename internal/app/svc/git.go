package svc

import (
	"context"
	"github.com/beldeveloper/app-lego/internal/app"
	"github.com/beldeveloper/app-lego/pkg/os"
	"github.com/beldeveloper/go-errors-context"
	"log"
	"regexp"
	"strings"
)

// NewGit creates a new instance of the git service.
func NewGit(reposDir app.ReposDir) app.VcsSvc {
	return Git{
		reposDir:       string(reposDir),
		remoteBranchRx: regexp.MustCompile("^([a-f0-9]+)\\s+refs/(heads|tags)/(.*)$"),
	}
}

// Git is a service that manages VCS.
type Git struct {
	reposDir       string
	remoteBranchRx *regexp.Regexp
}

// DownloadRepository to the directory.
func (s Git) DownloadRepository(ctx context.Context, r app.Repository) error {
	_, err := os.Exec(ctx, os.Cmd{
		Name: "git",
		Args: []string{"clone", r.Name, r.Alias},
		Dir:  s.reposDir,
		Log:  true,
	})
	return errors.WrapContext(err, errors.Context{
		Path:   "svc.Git.DownloadRepository",
		Params: errors.Params{"repository": r.ID},
	})
}

// Branches returns a list git branches and tags for the specific repository.
func (s Git) Branches(ctx context.Context, r app.Repository) ([]app.VcsBranch, error) {
	out, err := os.Exec(ctx, os.Cmd{
		Name: "git",
		Args: []string{"ls-remote"},
		Dir:  s.reposDir + "/" + r.Alias,
	})
	if err != nil {
		return nil, errors.WrapContext(err, errors.Context{
			Path:   "svc.Git.Branches.ls",
			Params: errors.Params{"repository": r.ID},
		})
	}
	rows := strings.Split(out, "\n")
	branches := make([]app.VcsBranch, 0, len(rows))
	var b app.VcsBranch
	for _, r := range rows {
		r := strings.TrimSpace(r)
		matches := s.remoteBranchRx.FindStringSubmatch(r)
		if len(matches) < 4 {
			continue
		}
		b.Hash = matches[1]
		b.Name = matches[3]
		switch matches[2] {
		case "heads":
			b.Type = app.BranchTypeHead
		case "tags":
			b.Type = app.BranchTypeTag
		default:
			continue
		}
		branches = append(branches, b)
	}
	return branches, nil
}

// SwitchBranch fetches git updates, resets the current branch, switches to a new one, and pulls branch updates.
func (s Git) SwitchBranch(ctx context.Context, r app.Repository, b app.Branch) error {
	_, err := os.Exec(ctx, os.Cmd{
		Name: "git",
		Args: []string{"fetch"},
		Dir:  s.reposDir + "/" + r.Alias,
		Log:  true,
	})
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Git.SwitchBranch.fetch",
			Params: errors.Params{"repository": r.ID},
		})
	}
	newBranch := b.Name
	if b.Type == app.BranchTypeTag {
		newBranch = "tags/" + b.Name
	}
	_, err = os.Exec(ctx, os.Cmd{
		Name: "git",
		Args: []string{"reset", "--hard"},
		Dir:  s.reposDir + "/" + r.Alias,
		Log:  true,
	})
	if err != nil {
		log.Println(errors.WrapContext(err, errors.Context{
			Path:   "svc.Git.SwitchBranch.reset",
			Params: errors.Params{"repository": r.ID},
		}))
	}
	_, err = os.Exec(ctx, os.Cmd{
		Name: "git",
		Args: []string{"checkout", newBranch},
		Dir:  s.reposDir + "/" + r.Alias,
		Log:  true,
	})
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Git.SwitchBranch.checkout",
			Params: errors.Params{"repository": r.ID, "branchName": b.Name},
		})
	}
	if b.Type == app.BranchTypeTag {
		return nil
	}
	_, err = os.Exec(ctx, os.Cmd{
		Name: "git",
		Args: []string{"reset", "--hard", "origin/" + b.Name},
		Dir:  s.reposDir + "/" + r.Alias,
		Log:  true,
	})
	if err != nil {
		log.Println(errors.WrapContext(err, errors.Context{
			Path:   "svc.Git.SwitchBranch.reset",
			Params: errors.Params{"repository": r.ID},
		}))
	}
	_, err = os.Exec(ctx, os.Cmd{
		Name: "git",
		Args: []string{"pull"},
		Dir:  s.reposDir + "/" + r.Alias,
		Log:  true,
	})
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Git.SwitchBranch.pull",
			Params: errors.Params{"repository": r.ID, "branch": b.ID},
		})
	}
	return nil
}
