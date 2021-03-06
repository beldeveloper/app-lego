package builder

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/app-lego/service/branch"
	"github.com/beldeveloper/app-lego/service/marshaller"
	"github.com/beldeveloper/app-lego/service/os"
	"github.com/beldeveloper/app-lego/service/repository"
	"github.com/beldeveloper/app-lego/service/variable"
	"github.com/beldeveloper/app-lego/service/vcs"
	"github.com/beldeveloper/go-errors-context"
	"log"
	goos "os"
	"strings"
	"sync"
)

// NewBuilder creates a new instance of the branch builder.
func NewBuilder(
	workDir model.FilePath,
	vcs vcs.Service,
	os os.Service,
	repositories repository.Service,
	branches branch.Service,
	variables variable.Service,
	dockerMarshaller marshaller.Service,
) Service {
	return Builder{
		reposDir:         string(workDir + "/" + model.RepositoriesDir),
		configDir:        string(workDir + "/" + model.ConfigDir),
		branchesDir:      string(workDir + "/" + model.BranchesDir),
		vcs:              vcs,
		os:               os,
		repositories:     repositories,
		branches:         branches,
		mux:              &sync.RWMutex{},
		queue:            make(map[uint64]bool),
		variables:        variables,
		dockerMarshaller: dockerMarshaller,
	}
}

// Builder is a service that is in charge of building the repository branch.
type Builder struct {
	reposDir         string
	configDir        string
	branchesDir      string
	vcs              vcs.Service
	os               os.Service
	repositories     repository.Service
	branches         branch.Service
	mux              *sync.RWMutex
	queue            map[uint64]bool
	variables        variable.Service
	dockerMarshaller marshaller.Service
}

// Enqueue puts the branch into building queue.
func (b Builder) Enqueue(ctx context.Context, branch model.Branch) error {
	branch.Status = model.BranchStatusEnqueued
	err := b.updateBranchStatus(ctx, branch)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "service.builder.Enqueue",
			Params: errors.Params{"branch": branch.ID, "status": branch.Status},
		})
	}
	b.toggleQueue(branch.ID, true)
	return nil
}

// Build reads the configuration from the repository and builds the brunch.
func (b Builder) Build(ctx context.Context, branch model.Branch) error {
	branch.Status = model.BranchStatusBuilding
	err := b.updateBranchStatus(ctx, branch)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "service.builder.Build: update status",
			Params: errors.Params{"branch": branch.ID, "status": branch.Status},
		})
	}
	_ = goos.RemoveAll(fmt.Sprintf("%s/%d", b.branchesDir, branch.ID))
	_ = goos.Mkdir(fmt.Sprintf("%s/%d", b.branchesDir, branch.ID), 0755)
	b.toggleQueue(branch.ID, false)
	step := b.prepareSteps(ctx, branch)
	for step != nil {
		if b.checkQueue(branch.ID) { // re-enqueued
			return model.ErrBuildCanceled
		}
		err = step.action()
		if err != nil {
			if errors.Is(err, model.ErrConfigurationNotFound) {
				branch.Status = model.BranchStatusSkipped
			} else {
				branch.Status = model.BranchStatusFailed
				err = errors.WrapContext(err, errors.Context{
					Path:   "service.builder.Build: run step",
					Params: errors.Params{"branch": branch.ID, "step": step.name},
				})
			}
			go func() {
				err := b.updateBranchStatus(ctx, branch)
				if err != nil {
					log.Println(errors.WrapContext(err, errors.Context{
						Path:   "service.builder.Build: update status",
						Params: errors.Params{"branch": branch.ID, "status": branch.Status},
					}))
				}
			}()
			return err
		}
		step = step.next
	}
	branch.Status = model.BranchStatusReady
	err = b.updateBranchStatus(ctx, branch)
	return errors.WrapContext(err, errors.Context{
		Path:   "service.builder.Build: update status",
		Params: errors.Params{"branch": branch.ID, "status": branch.Status},
	})
}

func (b Builder) toggleQueue(bID uint64, state bool) {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.queue[bID] = state
}

func (b Builder) checkQueue(bID uint64) bool {
	b.mux.RLock()
	defer b.mux.RUnlock()
	return b.queue[bID]
}

func (b Builder) updateBranchStatus(ctx context.Context, branch model.Branch) error {
	b.mux.Lock()
	defer b.mux.Unlock()
	err := b.branches.UpdateStatus(ctx, branch)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "service.builder.updateBranchStatus",
			Params: errors.Params{"branch": branch.ID, "status": branch.Status},
		})
	}
	return nil
}

func (b Builder) prepareSteps(ctx context.Context, branch model.Branch) *buildingStep {
	var r model.Repository
	var cfg model.BranchCfg
	var preDeployCmd, postDeployCmd []model.Cmd

	fetchRepositoryStep := buildingStep{
		name: "find repository",
		action: func() (err error) {
			r, err = b.repositories.FindByID(ctx, branch.RepositoryID)
			return errors.WrapContext(err, errors.Context{
				Path:   "service.builder.prepareSteps.fetchRepositoryStep",
				Params: errors.Params{"repository": branch.RepositoryID},
			})
		},
	}

	switchBranchStep := buildingStep{
		name: "switch branch",
		action: func() error {
			err := b.vcs.SwitchBranch(ctx, r, branch)
			return errors.WrapContext(err, errors.Context{
				Path:   "service.builder.prepareSteps.switchBranchStep",
				Params: errors.Params{"branch": branch.ID},
			})
		},
	}

	readConfigurationStep := buildingStep{name: "read build configuration"}

	finishStep := buildingStep{
		name: "finish",
		action: func() error {
			data, err := b.dockerMarshaller.Marshal(model.BranchComposeData{
				PreDeploy:       preDeployCmd,
				PostDeploy:      postDeployCmd,
				ComposeServices: cfg.Compose.Services,
			})
			if err != nil {
				return errors.WrapContext(err, errors.Context{Path: "service.builder.prepareSteps.finishStep: marshal"})
			}
			err = b.branches.SaveComposeData(ctx, branch, data)
			return errors.WrapContext(err, errors.Context{
				Path:   "service.builder.prepareSteps.finishStep: save compose data",
				Params: errors.Params{"branch": branch.ID},
			})
		},
	}

	readConfigurationStep.action = func() (err error) {
		cfg, err = b.readConfiguration(ctx, r, branch)
		if err != nil {
			return errors.WrapContext(err, errors.Context{
				Path:   "service.builder.prepareSteps.readConfigurationStep: read configuration",
				Params: errors.Params{"branch": branch.ID},
			})
		}
		preDeployCmd = cfg.Commands(cfg.PreDeploy)
		postDeployCmd = cfg.Commands(cfg.PostDeploy)
		currStep := &readConfigurationStep
		for _, cmd := range cfg.Commands(cfg.Build) {
			cmd := cmd
			cmd.Log = true
			if cmd.Dir == "" {
				cmd.Dir = b.reposDir + "/" + r.Alias
			} else if strings.HasPrefix(cmd.Dir, ".") {
				cmd.Dir = b.reposDir + "/" + r.Alias + "/" + cmd.Dir
			}
			step := buildingStep{
				name: "command: " + cmd.Name + " " + strings.Join(cmd.Args, " "),
				action: func() error {
					_, err := b.os.RunCmd(ctx, cmd)
					return errors.WrapContext(err, errors.Context{
						Path:   "service.builder.prepareSteps.cmd",
						Params: errors.Params{"cmd": cmd.Name, "args": cmd.Args, "env": cmd.Env, "dir": cmd.Dir},
					})
				},
				next: &finishStep,
			}
			currStep.next = &step
			currStep = &step
		}
		return nil
	}

	fetchRepositoryStep.next = &switchBranchStep
	switchBranchStep.next = &readConfigurationStep
	readConfigurationStep.next = &finishStep

	return &fetchRepositoryStep
}

func (b Builder) readConfiguration(ctx context.Context, r model.Repository, branch model.Branch) (model.BranchCfg, error) {
	var cfg model.BranchCfg
	cfgFile := vcs.DefaultCfgFile
	if r.CfgFile != "" {
		cfgFile = r.CfgFile
	}
	cfgFile = fmt.Sprintf("%s/%s/%s", b.reposDir, r.Alias, cfgFile)
	cfgData, err := b.os.ReadFile(ctx, cfgFile)
	if err != nil && errors.Is(err, goos.ErrNotExist) {
		cfgFile = fmt.Sprintf("%s/repositories/%s/%s", b.configDir, r.Alias, vcs.DefaultCfgFile)
		cfgData, err = b.os.ReadFile(ctx, cfgFile)
	}
	if err != nil {
		if errors.Is(err, goos.ErrNotExist) {
			err = model.ErrConfigurationNotFound
		}
		return cfg, errors.WrapContext(err, errors.Context{
			Path:   "service.builder.unmarshalConfiguration: read vcs configuration",
			Params: errors.Params{"branch": branch.ID},
		})
	}
	cfg.Variables, err = b.variables.ListFromSources(ctx, model.VariablesSources{
		Repository: r,
		Branch:     branch,
		CustomData: cfgData,
	})
	if err != nil {
		return cfg, errors.WrapContext(err, errors.Context{
			Path:   "service.builder.unmarshalConfiguration: list variables",
			Params: errors.Params{"repository": r.ID, "branch": branch.ID},
		})
	}
	cfgData, err = b.variables.Replace(ctx, cfgData, cfg.Variables)
	if err != nil {
		return cfg, errors.WrapContext(err, errors.Context{
			Path:   "service.builder.unmarshalConfiguration: replace variables",
			Params: errors.Params{"repository": r.ID, "branch": branch.ID},
		})
	}
	err = b.dockerMarshaller.Unmarshal(cfgData, &cfg)
	if err != nil {
		return cfg, errors.WrapContext(err, errors.Context{
			Path:   "service.builder.unmarshalConfiguration: unmarshal cfg",
			Params: errors.Params{"repository": r.ID, "branch": branch.ID},
		})
	}
	return cfg, nil
}
