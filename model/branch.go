package model

const (
	// BranchTypeHead defines the type for the repository head.
	BranchTypeHead = "head"
	// BranchTypeTag defines the type for the repository tag.
	BranchTypeTag = "tag"

	// BranchStatusPending defines the status that means the branch was recently added/updated and not enqueued yet.
	BranchStatusPending = "pending"
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
