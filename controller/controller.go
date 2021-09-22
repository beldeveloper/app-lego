package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/beldeveloper/app-lego/model"
	"github.com/beldeveloper/app-lego/service"
)

const (
	// AddRepositoryFrequency defines the frequency of the add repository job.
	AddRepositoryFrequency = time.Second * 10
	// SyncRepositoryFrequency defines the frequency of the repository sync job.
	SyncRepositoryFrequency = time.Second * 10
	// BuildBranchFrequency defines the frequency of the branch build job.
	BuildBranchFrequency = time.Second * 2
	// WatchDeploymentsFrequency defines the frequency of the watch deployments job.
	WatchDeploymentsFrequency = time.Second * 2
)

// NewController creates a new instance of the application controller.
func NewController(services service.Container) Controller {
	return Controller{services: services}
}

// Controller implements the application controller.
type Controller struct {
	services service.Container
}

// AddRepository adds new repository and puts int to pending download status.
func (c Controller) AddRepository(ctx context.Context, f model.FormAddRepository) (model.Repository, error) {
	f, err := c.services.Validation.AddRepository(ctx, f)
	if err != nil {
		if errors.Is(err, model.ErrBadInput) {
			return model.Repository{}, err
		}
		return model.Repository{}, fmt.Errorf("controller.AddRepository: error during validation: %w", err)
	}
	r, err := c.services.Repository.Add(ctx, model.Repository{
		Type:      f.Type,
		Alias:     f.Alias,
		Name:      f.Name,
		Status:    model.RepositoryStatusPending,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return r, fmt.Errorf("controller.AddRepository: coudn't add the repository: %w", err)
	}
	log.Printf("The repository #%d is added\n", r.ID)
	return r, nil
}

// Repositories returns the list of repositories.
func (c Controller) Repositories(ctx context.Context) ([]model.Repository, error) {
	return c.services.Repository.FindAll(ctx)
}

// Branches returns the list of branches.
func (c Controller) Branches(ctx context.Context) ([]model.Branch, error) {
	return c.services.Branches.FindAll(ctx)
}

// Deployments returns the list of deployments.
func (c Controller) Deployments(ctx context.Context) ([]model.Deployment, error) {
	return c.services.Deployment.FindAll(ctx)
}

// AddDeployment adds and enqueues new deployment.
func (c Controller) AddDeployment(ctx context.Context, f model.FormAddDeployment) (model.Deployment, error) {
	branches, err := c.services.Branches.FindByIDs(ctx, f.Branches)
	if err != nil {
		return model.Deployment{}, fmt.Errorf("controller.AddDeployment: find branches: %w", err)
	}
	d := model.Deployment{
		Status:    model.DeploymentStatusEnqueued,
		CreatedAt: time.Now(),
		Branches:  make([]model.DeploymentBranch, len(branches)),
	}
	for i, b := range branches {
		d.Branches[i] = model.DeploymentBranch{
			ID:   b.ID,
			Hash: b.Hash,
		}
	}
	d, err = c.services.Deployment.Add(ctx, d)
	if err != nil {
		return d, fmt.Errorf("controller.AddDeployment: add: %w", err)
	}
	log.Printf("The deployment #%d is requested\n", d.ID)
	return d, nil
}

// RebuildDeployment enqueues the existing deployment for rebuilding.
func (c Controller) RebuildDeployment(ctx context.Context, id uint64) (model.Deployment, error) {
	d, err := c.services.Deployment.FindByID(ctx, id)
	if err != nil {
		return d, fmt.Errorf("controller.RebuildDeployment: find by id: %w", err)
	}
	d.Status = model.DeploymentStatusEnqueued
	d, err = c.services.Deployment.Update(ctx, d)
	if err != nil {
		return d, fmt.Errorf("controller.RebuildDeployment: update: %w", err)
	}
	log.Printf("The deployment #%d is enqueued for rebuilding\n", d.ID)
	return d, nil
}

// CloseDeployment closes the existing deployment.
func (c Controller) CloseDeployment(ctx context.Context, id uint64) error {
	d, err := c.services.Deployment.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("controller.DeleteDeployment: find by id: %w", err)
	}
	d.Status = model.DeploymentStatusClosed
	d, err = c.services.Deployment.Update(ctx, d)
	if err != nil {
		return fmt.Errorf("controller.DeleteDeployment: update: %w", err)
	}
	log.Printf("The deployment #%d is closed\n", d.ID)
	return nil
}

// DownloadRepositoryJob is a job that downloads new repositories.
func (c Controller) DownloadRepositoryJob(ctx context.Context) {
	t := time.NewTicker(AddRepositoryFrequency)
	defer t.Stop()
	var r model.Repository
	var err error
	for {
		select {
		case <-t.C:
			r, err = c.services.Repository.FindPending(ctx)
			if err != nil {
				if !errors.Is(err, model.ErrNotFound) {
					log.Printf("controller.DownloadRepositoryJob: coudn't fetch the pending repository: %v\n", err)
				}
				break
			}
			r.Status = model.RepositoryStatusDownloading
			r, err = c.services.Repository.Update(ctx, r)
			if err != nil {
				log.Printf("controller.DownloadRepositoryJob: coudn't update repository status: %v; status = %s\n", err, r.Status)
				break
			}
			err = c.services.VCS.DownloadRepository(ctx, r)
			r.Status = model.RepositoryStatusReady
			if err != nil {
				r.Status = model.RepositoryStatusFailed
				log.Printf("controller.DownloadRepositoryJob: coudn't add the repository: %v; ID = %d\n", err, r.ID)
				break
			}
			r, err = c.services.Repository.Update(ctx, r)
			if err != nil {
				log.Printf("controller.DownloadRepositoryJob: coudn't update repository status: %v; status = %s\n", err, r.Status)
				break
			}
			log.Printf("The repository #%d is downloaded\n", r.ID)
		case <-ctx.Done():
			return
		}
	}
}

// SyncRepositoryJob is a job that pulls the outdated repositories and syncs them.
func (c Controller) SyncRepositoryJob(ctx context.Context) {
	t := time.NewTicker(SyncRepositoryFrequency)
	defer t.Stop()
	var r model.Repository
	var branches []model.Branch
	var err error
	for {
		select {
		case <-t.C:
			r, err = c.services.Repository.FindOutdated(ctx)
			if err != nil {
				if !errors.Is(err, model.ErrNotFound) {
					log.Printf("controller.SyncRepositoryJob: coudn't fetch the outdated repository: %v\n", err)
				}
				break
			}
			branches, err = c.services.VCS.Branches(ctx, r)
			if err != nil {
				log.Printf("controller.SyncRepositoryJob: coudn't get branches from VCS: %v; repository ID = %d\n", err, r.ID)
				break
			}
			branches, err = c.services.Branches.Sync(ctx, r, branches)
			if err != nil {
				log.Printf("controller.SyncRepositoryJob: coudn't sync branches: %v; repository ID = %d\n", err, r.ID)
				break
			}
			r.UpdatedAt = time.Now()
			r, err = c.services.Repository.Update(ctx, r)
			if err != nil {
				log.Printf("controller.SyncRepositoryJob: coudn't touch the repository: %v; repository ID = %d\n", err, r.ID)
				break
			}
			for _, b := range branches {
				err = c.services.Builder.Enqueue(ctx, b)
				if err != nil {
					log.Printf("controller.SyncRepositoryJob: coudn't enqueue branch: %v; branch ID = %d\n", err, b.ID)
					continue
				}
				log.Printf("The branch #%d is enqueued\n", b.ID)
			}
		case <-ctx.Done():
			return
		}
	}
}

// BuildBranchJob is a job that pulls the enqueued branches and builds them.
func (c Controller) BuildBranchJob(ctx context.Context) {
	t := time.NewTicker(BuildBranchFrequency)
	defer t.Stop()
	var b model.Branch
	var err error
	for {
		select {
		case <-t.C:
			b, err = c.services.Branches.FindEnqueued(ctx)
			if err != nil {
				if !errors.Is(err, model.ErrNotFound) {
					log.Printf("controller.BuildBranchJob: coudn't fetch the enqueued branch: %v\n", err)
				}
				break
			}
			err = c.services.Builder.Build(ctx, b)
			if err != nil {
				switch true {
				case errors.Is(err, model.ErrBuildCanceled):
					log.Printf("The branch #%d building is canceled\n", b.ID)
				case errors.Is(err, model.ErrConfigurationNotFound):
					log.Printf("The branch #%d building is skipped due to the configuration absence\n", b.ID)
				default:
					log.Printf("controller.BuildBranchJob: coudn't build the branch: %v; branch ID = %d\n", err, b.ID)
				}
				break
			}
			log.Printf("The branch #%d is built\n", b.ID)
		case <-ctx.Done():
			return
		}
	}
}

// WatchDeploymentsJob is a job that watches for the deployments state, builds, rebuilds, closes them.
func (c Controller) WatchDeploymentsJob(ctx context.Context) {
	t := time.NewTicker(WatchDeploymentsFrequency)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			err := c.services.Deployer.Run(ctx)
			if err != nil {
				log.Printf("controller.WatchDeploymentsJob: run deployer: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
