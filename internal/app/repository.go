package app

import (
	"context"
	"time"
)

const (
	// RepositoryTypeGit defines the type for Git repositories.
	RepositoryTypeGit = "git"

	// RepositoryStatusPending defines the status that means the repository was recently added and awaiting downloading.
	RepositoryStatusPending = "pending"
	// RepositoryStatusDownloading defines the status that means the repository is downloading.
	RepositoryStatusDownloading = "downloading"
	// RepositoryStatusReady defines the status that means the repository is ready.
	RepositoryStatusReady = "ready"
	// RepositoryStatusFailed defines the status that means the repository was added with an error.
	RepositoryStatusFailed = "failed"
)

// Repository is a model that represents a VCS repository.
type Repository struct {
	ID        uint64    `json:"id"`
	Type      string    `json:"type"`
	Alias     string    `json:"alias"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// FormAddRepository is a new repository form.
type FormAddRepository struct {
	Type  string `json:"type"`
	Alias string `json:"alias"`
	Name  string `json:"name"`
}

// RepositorySvc describes the repository service.
type RepositorySvc interface {
	List(context.Context) ([]Repository, error)
	Add(context.Context, FormAddRepository) (Repository, error)
	DownloadJob(ctx context.Context) error
	SyncJob(ctx context.Context) error
}

// RepositoryRepo describes interactions with the repository DB.
type RepositoryRepo interface {
	FindAll(ctx context.Context) ([]Repository, error)
	FindByID(ctx context.Context, id uint64) (Repository, error)
	FindPending(ctx context.Context) (Repository, error)
	FindOutdated(ctx context.Context) (Repository, error)
	Add(ctx context.Context, r Repository) (Repository, error)
	Update(ctx context.Context, r Repository) (Repository, error)
}
