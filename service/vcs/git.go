package vcs

import (
	"context"
	"errors"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	appOs "github.com/beldeveloper/app-lego/service/os"
	"github.com/beldeveloper/app-lego/service/variable"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
)

// NewGit creates a new instance of the Git VCS service.
func NewGit(workDir string, os appOs.Service, variable variable.Service) Git {
	return Git{
		workDir:        workDir,
		os:             os,
		variable:       variable,
		remoteBranchRx: regexp.MustCompile("^([a-f0-9]+)\\s+refs/(heads|tags)/(.*)$"),
	}
}

// Git implements the VCS service for Git.
type Git struct {
	workDir        string
	os             appOs.Service
	variable       variable.Service
	remoteBranchRx *regexp.Regexp
}

// DownloadRepository downloads the repository to the working directory.
func (g Git) DownloadRepository(ctx context.Context, r model.Repository) error {
	cmd := model.Cmd{
		Name: "git",
		Args: []string{"clone", r.Name, r.Alias},
		Dir:  g.workDir,
		Log:  true,
	}
	_, err := g.os.RunCmd(ctx, cmd)
	if err != nil {
		return fmt.Errorf("service.vcs.git.DownloadRepository: clone repository %d: %w", r.ID, err)
	}
	return nil
}

// Branches parses the branches and tags from the remote repository.
func (g Git) Branches(ctx context.Context, r model.Repository) ([]model.Branch, error) {
	out, err := g.os.RunCmd(ctx, model.Cmd{
		Name: "git",
		Args: []string{"ls-remote"},
		Dir:  g.workDir + "/" + r.Alias,
	})
	if err != nil {
		return nil, fmt.Errorf("service.vcs.git.Branches: ls remote branches: %w; repository ID = %d", err, r.ID)
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
		Args: []string{"checkout", b.Name},
		Dir:  g.workDir + "/" + r.Alias,
		Log:  true,
	})
	if err != nil {
		return fmt.Errorf("service.vcs.git.SwitchBranch: checkout: %w; branch ID = %d", err, b.ID)
	}
	_, err = g.os.RunCmd(ctx, model.Cmd{
		Name: "git",
		Args: []string{"pull"},
		Dir:  g.workDir + "/" + r.Alias,
		Log:  true,
	})
	if err != nil {
		return fmt.Errorf("service.vcs.git.SwitchBranch: pull: %w; branch ID = %d", err, b.ID)
	}
	return nil
}

// ReadConfiguration reads the configuration files from the specific branch.
func (g Git) ReadConfiguration(ctx context.Context, r model.Repository, b model.Branch) (model.BranchCfg, error) {
	var cfg model.BranchCfg
	f, err := os.OpenFile(fmt.Sprintf("%s/%s/app-lego.yml", g.workDir, r.Alias), os.O_RDONLY, 0755)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, model.ErrConfigurationNotFound
		}
		return cfg, fmt.Errorf("service.vcs.git.ReadConfiguration: open cfg file: %w; branch ID = %d", err, b.ID)
	}
	defer func() {
		err := f.Close()
		if err != nil {
			log.Printf("service.vcs.git.ReadConfiguration: close cfg file: %v; branch ID = %d", err, b.ID)
		}
	}()
	cfgData, err := ioutil.ReadAll(f)
	if err != nil {
		return cfg, fmt.Errorf("service.vcs.git.ReadConfiguration: read cfg file: %w; branch ID = %d", err, b.ID)
	}
	cfgData, err = g.variable.Replace(ctx, cfgData, model.Variables{Repository: r, Branch: b})
	if err != nil {
		return cfg, fmt.Errorf("service.vcs.git.ReadConfiguration: replace variables: %w; branch ID = %d", err, b.ID)
	}
	err = yaml.Unmarshal(cfgData, &cfg)
	if err != nil {
		return cfg, fmt.Errorf("service.vcs.git.ReadConfiguration: unmarshal cfg file: %w; branch ID = %d", err, b.ID)
	}
	return cfg, nil
}
