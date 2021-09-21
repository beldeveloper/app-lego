package model

import "time"

const (
	// DeploymentStatusEnqueued defines the status that means the deployment is recently added.
	DeploymentStatusEnqueued = "enqueued"
	// DeploymentStatusBuilding defines the status that means the application is building the deployment.
	DeploymentStatusBuilding = "building"
	// DeploymentStatusPendingRebuild defines the status that means the deployment is ready to be rebuilt.
	DeploymentStatusPendingRebuild = "pending_rebuild"
	// DeploymentStatusRebuilding defines the status that means the application is re-building the deployment.
	DeploymentStatusRebuilding = "rebuilding"
	// DeploymentStatusReady defines the status that means the deployment is built successfully.
	DeploymentStatusReady = "ready"
	// DeploymentStatusFailed defines the status that means the build attempt failed.
	DeploymentStatusFailed = "failed"
	// DeploymentStatusPendingClose defines the status that means the deployment is ready to be closed.
	DeploymentStatusPendingClose = "pending_close"
	// DeploymentStatusClosed defines the status that means the deployment is closed.
	DeploymentStatusClosed = "closed"
)

// Deployment is a model that represents a single deployment.
type Deployment struct {
	ID        uint64             `json:"id"`
	Status    string             `json:"status"`
	CreatedAt time.Time          `json:"createdAt"`
	Branches  []DeploymentBranch `json:"branches"`
}

// DeploymentBranch is a model that contains a snapshot of the branch data used in the particular deployment.
type DeploymentBranch struct {
	ID   uint64 `json:"id"`
	Hash string `json:"hash"`
}

// FormAddDeployment represents a form of new deployment.
type FormAddDeployment struct {
	Branches []uint64 `json:"branches"`
}
