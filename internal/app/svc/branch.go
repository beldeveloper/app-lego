package svc

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/internal/app"
	"github.com/beldeveloper/app-lego/internal/app/errtype"
	"github.com/beldeveloper/app-lego/pkg"
	"github.com/beldeveloper/go-errors-context"
	"log"
)

// NewBranch creates a new instance of the VCS branch service.
func NewBranch(
	vcsSvc app.VcsSvc,
	deploySvc app.DeploymentSvc,
	hookSvc pkg.HookSvc,
	branchRepo app.BranchRepo,
	repRepo app.RepositoryRepo,
) app.BranchSvc {
	return Branch{
		vcsSvc:     vcsSvc,
		deploySvc:  deploySvc,
		hookSvc:    hookSvc,
		branchRepo: branchRepo,
		repRepo:    repRepo,
	}
}

// Branch is a service that manages the VCS branches.
type Branch struct {
	vcsSvc     app.VcsSvc
	deploySvc  app.DeploymentSvc
	hookSvc    pkg.HookSvc
	branchRepo app.BranchRepo
	repRepo    app.RepositoryRepo
}

// List all branches.
func (s Branch) List(ctx context.Context) ([]app.Branch, error) {
	res, err := s.branchRepo.FindAll(ctx)
	return res, errors.WrapContext(err, errors.Context{Path: "svc.Branch.List.FindAll"})
}

// Sync all repository branches with VCS.
func (s Branch) Sync(ctx context.Context, r app.Repository) error {
	vcsBranches, err := s.vcsSvc.Branches(ctx, r)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Branch.Sync.Branches",
			Params: errors.Params{"repository": r.ID},
		})
	}
	old, err := s.branchRepo.FindByRepository(ctx, r)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Branch.Sync.FindByRepository",
			Params: errors.Params{"repository": r.ID},
		})
	}
	oldMap := make(map[string]app.Branch)
	for _, b := range old {
		oldMap[fmt.Sprintf("%s/%s", b.Type, b.Name)] = b
	}
	keepMap := make(map[uint64]bool)
	for _, b := range vcsBranches {
		oldBranch, exists := oldMap[fmt.Sprintf("%s/%s", b.Type, b.Name)]
		if !exists {
			_, err = s.branchRepo.Add(ctx, app.Branch{
				RepositoryID: r.ID,
				Type:         b.Type,
				Name:         b.Name,
				Hash:         b.Hash,
				Status:       app.BranchStatusEnqueued,
			})
			if err != nil {
				return errors.WrapContext(err, errors.Context{
					Path:   "svc.Branch.Sync.Add",
					Params: errors.Params{"branchName": b.Name, "branchType": b.Type, "branchHash": b.Hash},
				})
			}
			continue
		}
		keepMap[oldBranch.ID] = true
		if b.Hash == oldBranch.Hash ||
			oldBranch.Status == app.BranchStatusBuilding || oldBranch.Status == app.BranchStatusEnqueued {
			continue
		}
		oldBranch.Hash = b.Hash
		oldBranch.Status = app.BranchStatusEnqueued
		_, err = s.branchRepo.Update(ctx, oldBranch)
		if err != nil {
			return errors.WrapContext(err, errors.Context{
				Path:   "svc.Branch.Sync.Update",
				Params: errors.Params{"branch": oldBranch.ID},
			})
		}
	}
	del := make([]uint64, 0, len(old))
	for _, b := range old {
		if !keepMap[b.ID] {
			del = append(del, b.ID)
		}
	}
	err = s.branchRepo.DeleteByIDs(ctx, del)
	if err != nil {
		log.Println(errors.WrapContext(err, errors.Context{
			Path:   "svc.Branch.Sync.DeleteByIDs",
			Params: errors.Params{"ids": del},
		}))
	}
	err = s.hookSvc.CleanBranches(ctx, del)
	if err != nil {
		log.Println(errors.WrapContext(err, errors.Context{
			Path:   "svc.Branch.Sync.CleanBranches",
			Params: errors.Params{"ids": del},
		}))
	}
	return nil
}

// BuildJob fetches enqueued branches and builds them.
func (s Branch) BuildJob(ctx context.Context) error {
	b, err := s.branchRepo.FindEnqueued(ctx)
	if err != nil {
		if !errors.Is(err, errtype.ErrNotFound) {
			return errors.WrapContext(err, errors.Context{Path: "svc.Branch.BuildJob.FindEnqueued"})
		}
		return nil
	}
	r, err := s.repRepo.FindByID(ctx, b.RepositoryID)
	if err != nil {
		log.Println(errors.WrapContext(err, errors.Context{
			Path:   "svc.Branch.BuildJob.findRepo",
			Params: errors.Params{"repository": b.RepositoryID},
		}))
		b.Status = app.BranchStatusFailed
		errorMsg := fmt.Sprintf("Can't find repository id=%d; err=%v", b.RepositoryID, err)
		b.ErrorMsg = &errorMsg
		s.updateStatus(ctx, b)
		return nil
	}
	b.Status = app.BranchStatusBuilding
	if !s.updateStatus(ctx, b) {
		return nil
	}
	err = s.vcsSvc.SwitchBranch(ctx, r, b)
	if err != nil {
		b.Status = app.BranchStatusFailed
		errorMsg := fmt.Sprintf("Can't switch branch id=%d; err=%v", b.ID, err)
		b.ErrorMsg = &errorMsg
		s.updateStatus(ctx, b)
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Branch.BuildJob.SwitchBranch",
			Params: errors.Params{"branch": b.ID},
		})
	}
	buildRes, err := s.hookSvc.BuildBranch(ctx, pkg.HookBuildBranchReq{
		Repo: pkg.HookRepo{
			ID:    r.ID,
			Type:  r.Type,
			Alias: r.Alias,
		},
		Branch: pkg.HookBranch{
			ID:     b.ID,
			RepoID: b.RepositoryID,
			Type:   b.Type,
			Name:   b.Name,
			Hash:   b.Hash,
		},
	})
	if err != nil {
		b.Status = app.BranchStatusFailed
		errorMsg := err.Error()
		b.ErrorMsg = &errorMsg
		s.updateStatus(ctx, b)
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Branch.BuildJob.BuildBranch",
			Params: errors.Params{"branch": b.ID},
		})
	}
	switch buildRes.Status {
	case app.BranchStatusSkipped, app.BranchStatusReady:
		b.Status = buildRes.Status
	default:
		b.Status = app.BranchStatusFailed
		b.ErrorMsg = buildRes.ErrorMsg
		log.Printf("The branch #%d was not built, see details in hook handler; status=%s\n", b.ID, buildRes.Status)
	}
	if !s.updateStatus(ctx, b) || b.Status != app.BranchStatusReady {
		return nil
	}
	log.Printf("The branch #%d is built\n", b.ID)
	err = s.deploySvc.RebuildWithBranch(ctx, b)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Branch.BuildJob.RebuildWithBranch",
			Params: errors.Params{"branch": b.ID},
		})
	}
	return nil
}

func (s Branch) updateStatus(ctx context.Context, b app.Branch) bool {
	err := s.branchRepo.UpdateStatus(ctx, b)
	if err != nil {
		log.Println(errors.WrapContext(err, errors.Context{
			Path:   "svc.Branch.updateStatus",
			Params: errors.Params{"branch": b.ID, "status": b.Status},
		}))
		return false
	}
	return true
}
