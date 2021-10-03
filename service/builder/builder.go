package builder

import (
	"context"
	"errors"
	"fmt"
	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/app-lego/service/branch"
	"github.com/beldeveloper/app-lego/service/marshaller"
	"github.com/beldeveloper/app-lego/service/os"
	"github.com/beldeveloper/app-lego/service/repository"
	"github.com/beldeveloper/app-lego/service/vcs"
	"log"
	"strings"
	"sync"
)

// NewBuilder creates a new instance of the branch builder.
func NewBuilder(
	workDir string,
	vcs vcs.Service,
	os os.Service,
	repositories repository.Service,
	branches branch.Service,
	dockerMarshaller marshaller.Service,
) Builder {
	return Builder{
		workDir:          workDir,
		vcs:              vcs,
		os:               os,
		repositories:     repositories,
		branches:         branches,
		mux:              &sync.RWMutex{},
		queue:            make(map[uint64]bool),
		dockerMarshaller: dockerMarshaller,
	}
}

// Builder is a service that is in charge of building the repository branch.
type Builder struct {
	workDir          string
	vcs              vcs.Service
	os               os.Service
	repositories     repository.Service
	branches         branch.Service
	mux              *sync.RWMutex
	queue            map[uint64]bool
	dockerMarshaller marshaller.Service
}

// Enqueue puts the branch into building queue.
func (b Builder) Enqueue(ctx context.Context, branch model.Branch) error {
	branch.Status = model.BranchStatusEnqueued
	err := b.updateBranchStatus(ctx, branch)
	if err != nil {
		return err
	}
	b.toggleQueue(branch.ID, true)
	return nil
}

// Build reads the configuration from the repository and builds the brunch.
func (b Builder) Build(ctx context.Context, branch model.Branch) error {
	branch.Status = model.BranchStatusBuilding
	err := b.updateBranchStatus(ctx, branch)
	if err != nil {
		return err
	}
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
				err = fmt.Errorf(
					"service.builder.Build: coudn't execute step: %w; branch ID = %d; step = %s",
					err,
					branch.ID,
					step.name,
				)
			}
			go func() {
				err := b.updateBranchStatus(ctx, branch)
				if err != nil {
					log.Println(err)
				}
			}()
			return err
		}
		step = step.next
	}
	branch.Status = model.BranchStatusReady
	return b.updateBranchStatus(ctx, branch)
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
		return fmt.Errorf(
			"service.builder.setBranchStatus: coudn't set branch status to %s: %w; branch ID = %d\n",
			branch.Status,
			err,
			branch.ID,
		)
	}
	return nil
}

func (b Builder) prepareSteps(ctx context.Context, branch model.Branch) *buildingStep {
	var r model.Repository
	var cfg model.BranchCfg

	fetchRepositoryStep := buildingStep{
		name: "find repository",
		action: func() (err error) {
			r, err = b.repositories.FindByID(ctx, branch.RepositoryID)
			return err
		},
	}

	switchBranchStep := buildingStep{
		name: "switch branch",
		action: func() error {
			return b.vcs.SwitchBranch(ctx, r, branch)
		},
	}

	readConfigurationStep := buildingStep{name: "read build configuration"}

	finishStep := buildingStep{
		name: "finish",
		action: func() error {
			data, err := b.dockerMarshaller.Marshal(cfg.Compose)
			if err != nil {
				return err
			}
			return b.branches.SaveComposeData(ctx, branch, data)
		},
	}

	readConfigurationStep.action = func() (err error) {
		cfg, err = b.vcs.ReadConfiguration(ctx, r, branch)
		if err != nil {
			return err
		}
		currStep := &readConfigurationStep
		for _, cmd := range cfg.Commands() {
			cmd := cmd
			cmd.Log = true
			if strings.HasPrefix(cmd.Dir, ".") {
				cmd.Dir = b.workDir + "/" + r.Alias + "/" + cmd.Dir
			}
			step := buildingStep{
				name: "command: " + cmd.Name + " " + strings.Join(cmd.Args, " "),
				action: func() error {
					_, err := b.os.RunCmd(ctx, cmd)
					return err
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
