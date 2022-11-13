package app

import "context"

// ReposDir is a data type for storing the repositories' directory, used for DI.
type ReposDir string

// VcsBranch is a model that represents a repository branch in VCS.
type VcsBranch struct {
	Type string
	Name string
	Hash string
}

// VcsSvc describes the version control service.
type VcsSvc interface {
	DownloadRepository(ctx context.Context, r Repository) error
	Branches(ctx context.Context, r Repository) ([]VcsBranch, error)
	SwitchBranch(ctx context.Context, r Repository, b Branch) error
}
