package vcs

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/app-lego/service/marshaller"
	appOs "github.com/beldeveloper/app-lego/service/os"
	"github.com/beldeveloper/app-lego/service/variable"
	"github.com/beldeveloper/go-errors-context"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

// NewGit creates a new instance of the Git VCS service.
func NewGit(workDir string, os appOs.Service, variable variable.Service, cfgMarshaller marshaller.Service) Git {
	return Git{
		workDir:        workDir,
		os:             os,
		variable:       variable,
		cfgMarshaller:  cfgMarshaller,
		remoteBranchRx: regexp.MustCompile("^([a-f0-9]+)\\s+refs/(heads|tags)/(.*)$"),
	}
}

// Git implements the VCS service for Git.
type Git struct {
	workDir        string
	os             appOs.Service
	variable       variable.Service
	cfgMarshaller  marshaller.Service
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
	_, err = g.os.RunCmd(ctx, model.Cmd{
		Name: "git",
		Args: []string{"checkout", b.Name},
		Dir:  g.workDir + "/" + r.Alias,
		Log:  true,
	})
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "service.vcs.git.SwitchBranch: checkout",
			Params: errors.Params{"repository": r.ID, "branchName": b.Name},
		})
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

// ReadConfiguration reads the configuration files from the specific branch.
func (g Git) ReadConfiguration(ctx context.Context, r model.Repository, b model.Branch) (model.BranchCfg, error) {
	var cfg model.BranchCfg
	f, err := os.OpenFile(fmt.Sprintf("%s/%s/app-lego.yml", g.workDir, r.Alias), os.O_RDONLY, 0755)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = model.ErrConfigurationNotFound
		}
		return cfg, errors.WrapContext(err, errors.Context{
			Path:   "service.vcs.git.ReadConfiguration: open file",
			Params: errors.Params{"repository": r.ID, "branch": b.ID},
		})
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Println(errors.WrapContext(err, errors.Context{
				Path:   "service.vcs.git.ReadConfiguration: close file",
				Params: errors.Params{"repository": r.ID, "branch": b.ID},
			}))
		}
	}()
	cfgData, err := ioutil.ReadAll(f)
	if err != nil {
		return cfg, errors.WrapContext(err, errors.Context{
			Path:   "service.vcs.git.ReadConfiguration: read file",
			Params: errors.Params{"repository": r.ID, "branch": b.ID},
		})
	}
	cfg.Variables, err = g.variable.ListFromSources(ctx, model.VariablesSources{Repository: r, Branch: b, CustomData: cfgData})
	if err != nil {
		return cfg, errors.WrapContext(err, errors.Context{
			Path:   "service.vcs.git.ReadConfiguration: list variables",
			Params: errors.Params{"repository": r.ID, "branch": b.ID},
		})
	}
	cfgData, err = g.variable.Replace(ctx, cfgData, cfg.Variables)
	if err != nil {
		return cfg, errors.WrapContext(err, errors.Context{
			Path:   "service.vcs.git.ReadConfiguration: replace variables",
			Params: errors.Params{"repository": r.ID, "branch": b.ID},
		})
	}
	err = g.cfgMarshaller.Unmarshal(cfgData, &cfg)
	if err != nil {
		return cfg, errors.WrapContext(err, errors.Context{
			Path:   "service.vcs.git.ReadConfiguration: unmarshal cfg",
			Params: errors.Params{"repository": r.ID, "branch": b.ID},
		})
	}
	return cfg, nil
}
