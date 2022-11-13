package app

import (
	"context"
)

const (
	// BranchTypeHead defines the type for the repository head.
	BranchTypeHead = "head"
	// BranchTypeTag defines the type for the repository tag.
	BranchTypeTag = "tag"

	// BranchStatusEnqueued defines the status that means the branch was updated and is ready to be built.
	BranchStatusEnqueued = "enqueued"
	// BranchStatusBuilding defines the status that means the application is building the branch.
	BranchStatusBuilding = "building"
	// BranchStatusReady defines the status that means the branch is up-to-date and built.
	BranchStatusReady = "ready"
	// BranchStatusFailed defines the status that means the build attempt failed.
	BranchStatusFailed = "failed"
	// BranchStatusSkipped defines the status that means the branch shouldn't be built.
	BranchStatusSkipped = "skipped"
)

// Branch is a model that represents a repository branch.
type Branch struct {
	ID           uint64 `json:"id"`
	RepositoryID uint64 `json:"repositoryId"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	Hash         string `json:"hash"`
	Status       string `json:"status"`
}

// BranchSvc describes the branch service.
type BranchSvc interface {
	List(context.Context) ([]Branch, error)
	Sync(ctx context.Context, r Repository) error
	BuildJob(ctx context.Context) error
}

// BranchRepo describes interactions with the branch DB.
type BranchRepo interface {
	FindAll(ctx context.Context) ([]Branch, error)
	FindByIDs(ctx context.Context, ids []uint64) ([]Branch, error)
	FindByID(ctx context.Context, id uint64) (Branch, error)
	FindByRepository(ctx context.Context, r Repository) ([]Branch, error)
	FindEnqueued(ctx context.Context) (Branch, error)
	Add(ctx context.Context, b Branch) (Branch, error)
	Update(ctx context.Context, b Branch) (Branch, error)
	UpdateStatus(ctx context.Context, b Branch) error
	DeleteByIDs(ctx context.Context, ids []uint64) error
}
