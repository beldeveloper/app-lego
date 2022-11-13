package svc

import (
	"context"
	"fmt"
	"github.com/beldeveloper/app-lego/internal/app"
	"github.com/beldeveloper/app-lego/internal/app/errtype"
	"github.com/beldeveloper/go-errors-context"
	"log"
	"strings"
	"time"
)

// NewRepository creates a new instance of the VCS repository service.
func NewRepository(
	vcsSvc app.VcsSvc,
	branchSvc app.BranchSvc,
	repo app.RepositoryRepo,
) app.RepositorySvc {
	return Repository{
		vcsSvc:    vcsSvc,
		branchSvc: branchSvc,
		repo:      repo,
	}
}

// Repository is a service that manages the VCS repositories.
type Repository struct {
	vcsSvc    app.VcsSvc
	branchSvc app.BranchSvc
	repo      app.RepositoryRepo
}

// List all repositories.
func (s Repository) List(ctx context.Context) ([]app.Repository, error) {
	res, err := s.repo.FindAll(ctx)
	return res, errors.WrapContext(err, errors.Context{Path: "svc.Repository.List.FindAll"})
}

// Add new repository.
func (s Repository) Add(ctx context.Context, f app.FormAddRepository) (app.Repository, error) {
	f, err := s.validateAddForm(f)
	if err != nil {
		return app.Repository{}, errors.WrapContext(err, errors.Context{Path: "svc.Repository.Add.validateAddForm"})
	}
	r, err := s.repo.Add(ctx, app.Repository{
		Type:      f.Type,
		Alias:     f.Alias,
		Name:      f.Name,
		Status:    app.RepositoryStatusPending,
		UpdatedAt: time.Now().Add(-time.Hour), // this way it will have a high priority for branches sync
	})
	if err != nil {
		return r, errors.WrapContext(err, errors.Context{Path: "svc.Repository.Add.Add"})
	}
	log.Printf("The repository #%d is added\n", r.ID)
	return r, nil
}

// DownloadJob looks for recently added repositories and downloads them.
func (s Repository) DownloadJob(ctx context.Context) error {
	r, err := s.repo.FindPending(ctx)
	if err != nil {
		if !errors.Is(err, errtype.ErrNotFound) {
			return errors.WrapContext(err, errors.Context{Path: "svc.Repository.DownloadJob.FindPending"})
		}
		return nil
	}
	r.Status = app.RepositoryStatusDownloading
	r, err = s.repo.Update(ctx, r)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Repository.DownloadJob.preUpdate",
			Params: errors.Params{"repository": r.ID, "status": r.Status},
		})
	}
	err = s.vcsSvc.DownloadRepository(ctx, r)
	r.Status = app.RepositoryStatusReady
	if err != nil {
		r.Status = app.RepositoryStatusFailed
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Repository.DownloadJob.DownloadRepository",
			Params: errors.Params{"repository": r.ID},
		})
	}
	r, err = s.repo.Update(ctx, r)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Repository.DownloadJob.postUpdate",
			Params: errors.Params{"repository": r.ID, "status": r.Status},
		})
	}
	log.Printf("The repository #%d is downloaded\n", r.ID)
	return nil
}

// SyncJob pulls the updates for the repositories.
func (s Repository) SyncJob(ctx context.Context) error {
	r, err := s.repo.FindOutdated(ctx)
	if err != nil {
		if !errors.Is(err, errtype.ErrNotFound) {
			return errors.WrapContext(err, errors.Context{Path: "svc.Repository.SyncJob.FindOutdated"})
		}
		return nil
	}
	err = s.branchSvc.Sync(ctx, r)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Repository.SyncJob.Sync",
			Params: errors.Params{"repository": r.ID},
		})
	}
	r.UpdatedAt = time.Now()
	r, err = s.repo.Update(ctx, r)
	if err != nil {
		return errors.WrapContext(err, errors.Context{
			Path:   "svc.Repository.SyncJob.Update",
			Params: errors.Params{"repository": r.ID},
		})
	}
	return nil
}

func (s Repository) validateAddForm(f app.FormAddRepository) (app.FormAddRepository, error) {
	if f.Type != app.RepositoryTypeGit {
		return f, fmt.Errorf("%w: repository type is invalid; allowed values: %s", errtype.ErrBadInput, app.RepositoryTypeGit)
	}
	f.Alias = strings.TrimSpace(f.Alias)
	f.Name = strings.TrimSpace(f.Name)
	if f.Alias == "" {
		return f, fmt.Errorf("%w: repository alias must not be empty", errtype.ErrBadInput)
	}
	if f.Name == "" {
		return f, fmt.Errorf("%w: repository name must not be empty", errtype.ErrBadInput)
	}
	return f, nil
}
