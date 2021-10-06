package controller

import (
	"context"
	"github.com/beldeveloper/go-errors-context"
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
		return model.Repository{}, errors.WrapContext(err, errors.Context{Path: "controller.AddRepository: validate"})
	}
	r, err := c.services.Repository.Add(ctx, model.Repository{
		Type:      f.Type,
		Alias:     f.Alias,
		Name:      f.Name,
		Status:    model.RepositoryStatusPending,
		UpdatedAt: time.Now(),
	})
	if err != nil {
		return r, errors.WrapContext(err, errors.Context{Path: "controller.AddRepository: add"})
	}
	log.Printf("The repository #%d is added\n", r.ID)
	return r, nil
}

// Repositories returns the list of repositories.
func (c Controller) Repositories(ctx context.Context) ([]model.Repository, error) {
	res, err := c.services.Repository.FindAll(ctx)
	return res, errors.WrapContext(err, errors.Context{Path: "controller.Repositories: find"})
}

// Branches returns the list of branches.
func (c Controller) Branches(ctx context.Context) ([]model.Branch, error) {
	res, err := c.services.Branches.FindAll(ctx)
	return res, errors.WrapContext(err, errors.Context{Path: "controller.Branches: find"})
}

// Deployments returns the list of deployments.
func (c Controller) Deployments(ctx context.Context) ([]model.Deployment, error) {
	res, err := c.services.Deployment.FindAll(ctx)
	return res, errors.WrapContext(err, errors.Context{Path: "controller.Deployments: find"})
}

// AddDeployment adds and enqueues new deployment.
func (c Controller) AddDeployment(ctx context.Context, f model.FormAddDeployment) (model.Deployment, error) {
	branches, err := c.services.Branches.FindByIDs(ctx, f.Branches)
	if err != nil {
		return model.Deployment{}, errors.WrapContext(err, errors.Context{
			Path:   "controller.AddDeployment: find branches",
			Params: errors.Params{"branches": f.Branches},
		})
	}
	d := model.Deployment{
		Status:      model.DeploymentStatusEnqueued,
		CreatedAt:   time.Now(),
		AutoRebuild: f.AutoRebuild,
		Branches:    make([]model.DeploymentBranch, len(branches)),
	}
	for i, b := range branches {
		d.Branches[i] = model.DeploymentBranch{
			ID:   b.ID,
			Hash: b.Hash,
		}
	}
	d, err = c.services.Deployment.Add(ctx, d)
	if err != nil {
		return d, errors.WrapContext(err, errors.Context{Path: "controller.AddDeployment: add"})
	}
	log.Printf("The deployment #%d is requested\n", d.ID)
	return d, nil
}

// RebuildDeployment enqueues the existing deployment for rebuilding.
func (c Controller) RebuildDeployment(ctx context.Context, id uint64) (model.Deployment, error) {
	d, err := c.services.Deployment.FindByID(ctx, id)
	if err != nil {
		return d, errors.WrapContext(err, errors.Context{
			Path:   "controller.RebuildDeployment: find",
			Params: errors.Params{"deployment": id},
		})
	}
	d.Status = model.DeploymentStatusEnqueued
	d, err = c.services.Deployment.Update(ctx, d)
	if err != nil {
		return d, errors.WrapContext(err, errors.Context{
			Path:   "controller.RebuildDeployment: update",
			Params: errors.Params{"deployment": id},
		})
	}
	log.Printf("The deployment #%d is enqueued for rebuilding\n", d.ID)
	return d, nil
}

// CloseDeployment closes the existing deployment.
func (c Controller) CloseDeployment(ctx context.Context, id uint64) error {
	d, err := c.services.Deployment.FindByID(ctx, id)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "controller.CloseDeployment: find",
			Params: errors.Params{"deployment": id},
		})
	}
	d.Status = model.DeploymentStatusClosed
	d, err = c.services.Deployment.Update(ctx, d)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "controller.CloseDeployment: update",
			Params: errors.Params{"deployment": id},
		})
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
					log.Println(errors.WrapContext(err, errors.Context{Path: "controller.DownloadRepositoryJob: find pending"}))
				}
				break
			}
			r.Status = model.RepositoryStatusDownloading
			r, err = c.services.Repository.Update(ctx, r)
			if err != nil {
				log.Println(errors.WrapContext(err, errors.Context{
					Path:   "controller.DownloadRepositoryJob: pre-update",
					Params: errors.Params{"repository": r.ID, "status": r.Status},
				}))
				break
			}
			err = c.services.VCS.DownloadRepository(ctx, r)
			r.Status = model.RepositoryStatusReady
			if err != nil {
				r.Status = model.RepositoryStatusFailed
				log.Println(errors.WrapContext(err, errors.Context{
					Path:   "controller.DownloadRepositoryJob: download",
					Params: errors.Params{"repository": r.ID},
				}))
				break
			}
			r, err = c.services.Repository.Update(ctx, r)
			if err != nil {
				log.Println(errors.WrapContext(err, errors.Context{
					Path:   "controller.DownloadRepositoryJob: post-update",
					Params: errors.Params{"repository": r.ID, "status": r.Status},
				}))
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
					log.Println(errors.WrapContext(err, errors.Context{Path: "controller.SyncRepositoryJob: find outdated"}))
				}
				break
			}
			branches, err = c.services.VCS.Branches(ctx, r)
			if err != nil {
				log.Println(errors.WrapContext(err, errors.Context{
					Path:   "controller.SyncRepositoryJob: branches",
					Params: errors.Params{"repository": r.ID},
				}))
				break
			}
			branches, err = c.services.Branches.Sync(ctx, r, branches)
			if err != nil {
				log.Println(errors.WrapContext(err, errors.Context{
					Path:   "controller.SyncRepositoryJob: sync",
					Params: errors.Params{"repository": r.ID},
				}))
				break
			}
			r.UpdatedAt = time.Now()
			r, err = c.services.Repository.Update(ctx, r)
			if err != nil {
				log.Println(errors.WrapContext(err, errors.Context{
					Path:   "controller.SyncRepositoryJob: update",
					Params: errors.Params{"repository": r.ID},
				}))
				break
			}
			for _, b := range branches {
				err = c.services.Builder.Enqueue(ctx, b)
				if err != nil {
					log.Println(errors.WrapContext(err, errors.Context{
						Path:   "controller.SyncRepositoryJob: enqueue",
						Params: errors.Params{"branch": b.ID},
					}))
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
					log.Println(errors.WrapContext(err, errors.Context{Path: "controller.BuildBranchJob: find enqueued"}))
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
					log.Println(errors.WrapContext(err, errors.Context{
						Path:   "controller.BuildBranchJob: build",
						Params: errors.Params{"branch": b.ID},
					}))
				}
				break
			}
			go func(b model.Branch) {
				err := c.services.Deployer.AutoRebuild(ctx, b)
				if err != nil {
					log.Println(errors.WrapContext(err, errors.Context{
						Path:   "controller.BuildBranchJob: auto-rebuild",
						Params: errors.Params{"branch": b.ID},
					}))
				}
			}(b)
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
				log.Println(errors.WrapContext(err, errors.Context{Path: "controller.WatchDeploymentsJob: run"}))
			}
		case <-ctx.Done():
			return
		}
	}
}
