package svc

import (
	"context"
	"github.com/beldeveloper/app-lego/internal/app"
	"github.com/beldeveloper/app-lego/pkg"
	"github.com/beldeveloper/go-errors-context"
	"log"
	"time"
)

// NewDeployment creates a new instance of the deployments service.
func NewDeployment(
	hookSvc pkg.HookSvc,
	deployRepo app.DeploymentRepo,
	branchRepo app.BranchRepo,
	repRepo app.RepositoryRepo,
) app.DeploymentSvc {
	return Deployment{
		hookSvc:    hookSvc,
		deployRepo: deployRepo,
		branchRepo: branchRepo,
		repRepo:    repRepo,
	}
}

// Deployment is a service that manages the deployments.
type Deployment struct {
	hookSvc    pkg.HookSvc
	deployRepo app.DeploymentRepo
	branchRepo app.BranchRepo
	repRepo    app.RepositoryRepo
}

// List returns non-closed deployments.
func (s Deployment) List(ctx context.Context) ([]app.Deployment, error) {
	res, err := s.deployRepo.FindAll(ctx)
	return res, errors.WrapContext(err, errors.Context{Path: "svc.Deployment.List.FindAll"})
}

// Add new deployment.
func (s Deployment) Add(ctx context.Context, f app.FormAddDeployment) (app.Deployment, error) {
	branches, err := s.branchRepo.FindByIDs(ctx, f.Branches)
	if err != nil {
		return app.Deployment{}, errors.WrapContext(err, errors.Context{
			Path:   "svc.Deployment.Add.FindByIDs",
			Params: errors.Params{"branches": f.Branches},
		})
	}
	d := app.Deployment{
		Status:      app.DeploymentStatusEnqueued,
		CreatedAt:   time.Now(),
		AutoRebuild: f.AutoRebuild,
		Branches:    make([]app.DeploymentBranch, len(branches)),
	}
	for i, b := range branches {
		d.Branches[i] = app.DeploymentBranch{
			ID:   b.ID,
			Hash: b.Hash,
		}
	}
	d, err = s.deployRepo.Add(ctx, d)
	if err != nil {
		return d, errors.WrapContext(err, errors.Context{Path: "svc.Deployment.Add.Add"})
	}
	log.Printf("The deployment #%d is requested\n", d.ID)
	return d, nil
}

// Rebuild deployment by ID.
func (s Deployment) Rebuild(ctx context.Context, f app.FormReDeployment) (app.Deployment, error) {
	d, err := s.deployRepo.FindByID(ctx, f.ID)
	if err != nil {
		return d, errors.WrapContext(err, errors.Context{
			Path:   "svc.Deployment.Rebuild.FindByID",
			Params: errors.Params{"deployment": f.ID},
		})
	}
	branches, err := s.branchRepo.FindByIDs(ctx, f.Branches)
	if err != nil {
		return app.Deployment{}, errors.WrapContext(err, errors.Context{
			Path:   "svc.Deployment.Add.FindByIDs",
			Params: errors.Params{"branches": f.Branches},
		})
	}
	d.Branches = make([]app.DeploymentBranch, len(branches))
	for i, b := range branches {
		d.Branches[i] = app.DeploymentBranch{
			ID:   b.ID,
			Hash: b.Hash,
		}
	}
	d.Status = app.DeploymentStatusEnqueued
	d, err = s.deployRepo.Update(ctx, d)
	if err != nil {
		return d, errors.WrapContext(err, errors.Context{
			Path:   "svc.Deployment.Rebuild.Update",
			Params: errors.Params{"deployment": d.ID},
		})
	}
	log.Printf("The deployment #%d is enqueued for rebuilding\n", d.ID)
	return d, nil
}

// RebuildWithBranch rebuilds all deployment that are attached to the specific branch.
func (s Deployment) RebuildWithBranch(ctx context.Context, b app.Branch) error {
	deployments, err := s.deployRepo.FindForAutoRebuild(ctx, b)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Deployment.RebuildWithBranch.FindForAutoRebuild",
			Params: errors.Params{"branch": b.ID},
		})
	}
	for _, d := range deployments {
		d.Status = app.DeploymentStatusEnqueued
		d, err = s.deployRepo.Update(ctx, d)
		if err != nil {
			log.Println(errors.WrapContext(err, errors.Context{
				Path:   "svc.Deployment.RebuildWithBranch.Update",
				Params: errors.Params{"deployment": d.ID},
			}))
			continue
		}
		log.Printf("Deployment #%d is enqueued for auto-rebuilding\n", d.ID)
	}
	return nil
}

// Close the deployment.
func (s Deployment) Close(ctx context.Context, id uint64) error {
	d, err := s.deployRepo.FindByID(ctx, id)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Deployment.Close.FindByID",
			Params: errors.Params{"deployment": id},
		})
	}
	d.Status = app.DeploymentStatusClosed
	d, err = s.deployRepo.Update(ctx, d)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Deployment.Close.Update",
			Params: errors.Params{"deployment": id},
		})
	}
	log.Printf("The deployment #%d is closed\n", d.ID)
	return nil
}

// WatchJob initiates redeployment if new deployment appears.
func (s Deployment) WatchJob(ctx context.Context) error {
	deployments, err := s.deployRepo.FindAll(ctx)
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "svc.Deployment.WatchJob.findDeployments"})
	}
	repos, err := s.repRepo.FindAll(ctx)
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "svc.Deployment.WatchJob.findRepositories"})
	}
	repoMap := make(map[uint64]app.Repository)
	for _, r := range repos {
		repoMap[r.ID] = r
	}
	branches, err := s.branchRepo.FindAll(ctx)
	if err != nil {
		return errors.WrapContext(err, errors.Context{Path: "svc.Deployment.WatchJob.findBranches"})
	}
	branchMap := make(map[uint64]app.Branch)
	for _, b := range branches {
		branchMap[b.ID] = b
	}
	makeHook := func(d app.Deployment, upd bool) pkg.HookDeployment {
		h := pkg.HookDeployment{ID: d.ID, Updated: upd, Branches: make(map[string]pkg.HookBranch, len(d.Branches))}
		for _, db := range d.Branches {
			branch := branchMap[db.ID]
			hash := db.Hash
			if upd {
				hash = branch.Hash
			}
			h.Branches[repoMap[branch.RepositoryID].Alias] = pkg.HookBranch{
				ID:     branch.ID,
				RepoID: branch.RepositoryID,
				Type:   branch.Type,
				Name:   branch.Name,
				Hash:   hash,
			}
		}
		return h
	}
	hookReq := pkg.HookDeployReq{
		Repos:       make([]pkg.HookRepo, 0, len(repos)),
		Deployments: make([]pkg.HookDeployment, 0, len(deployments)),
	}
	deployMap := make(map[uint64]app.Deployment, len(deployments))
	var upd bool
	for _, d := range deployments {
		if d.Status == app.DeploymentStatusBuilding {
			// skip if another deployment is in progress
			return nil
		}
		if d.Status == app.DeploymentStatusEnqueued {
			upd = true
			deployMap[d.ID] = d
			hookReq.Deployments = append(hookReq.Deployments, makeHook(d, true))
			continue
		}
		if d.Status == app.DeploymentStatusReady {
			for _, b := range d.Branches {
				if _, exists := branchMap[b.ID]; !exists {
					d.Status = app.DeploymentStatusClosed
					d, err = s.deployRepo.Update(ctx, d)
					if err != nil {
						log.Println(errors.WrapContext(err, errors.Context{
							Path:   "svc.Deployment.WatchJob.closeDeployment",
							Params: errors.Params{"deployment": d.ID},
						}))
					}
					break
				}
			}
			if d.Status == app.DeploymentStatusReady {
				hookReq.Deployments = append(hookReq.Deployments, makeHook(d, false))
			}
			continue
		}
	}
	if !upd {
		return nil
	}
	for _, r := range repos {
		hookReq.Repos = append(hookReq.Repos, pkg.HookRepo{
			ID:    r.ID,
			Type:  r.Type,
			Alias: r.Alias,
		})
	}
	err = s.massUpdateStatus(ctx, deployMap, app.DeploymentStatusBuilding, nil)
	if err != nil {
		return err
	}
	deployRes, err := s.hookSvc.Deploy(ctx, hookReq)
	if err != nil {
		errMsg := err.Error()
		err = s.massUpdateStatus(ctx, deployMap, app.DeploymentStatusFailed, &errMsg)
		if err != nil {
			log.Println(err)
		}
		return errors.WrapContext(err, errors.Context{Path: "svc.Deployment.WatchJob.Deploy"})
	}
	for id, status := range deployRes.Statuses {
		d, exists := deployMap[id]
		if !exists {
			continue
		}
		switch status.Status {
		case app.DeploymentStatusReady:
			d.Status = status.Status
			s.updateHashes(d, branchMap)
		default:
			d.Status = app.DeploymentStatusFailed
			d.ErrorMsg = status.ErrorMsg
			log.Printf("The deployment #%d was not deployed, see details in hook handler; status=%s\n", d.ID, status)
		}
		_, err = s.deployRepo.Update(ctx, d)
		if err != nil {
			log.Println(errors.WrapContext(err, errors.Context{
				Path:   "svc.WatchJob.Update",
				Params: errors.Params{"deployment": d.ID, "status": status},
			}))
		}
	}
	return nil
}

func (s Deployment) massUpdateStatus(ctx context.Context, deploys map[uint64]app.Deployment, status string, errorMsg *string) error {
	var err error
	for _, d := range deploys {
		d.Status = status
		d.ErrorMsg = errorMsg
		_, err = s.deployRepo.Update(ctx, d)
		if err != nil {
			return errors.WrapContext(err, errors.Context{
				Path:   "svc.Deployment.massUpdateStatus",
				Params: errors.Params{"deployment": d.ID, "status": status},
			})
		}
	}
	return nil
}

func (s Deployment) updateHashes(d app.Deployment, branchMap map[uint64]app.Branch) {
	for i, b := range d.Branches {
		d.Branches[i].Hash = branchMap[b.ID].Hash
	}
}
