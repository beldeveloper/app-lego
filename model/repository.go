package model

import "time"

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
