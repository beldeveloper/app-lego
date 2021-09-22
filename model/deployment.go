package model

import "time"

const (
	// DeploymentStatusEnqueued defines the status that means the deployment is ready to be built.
	DeploymentStatusEnqueued = "enqueued"
	// DeploymentStatusBuilding defines the status that means the application is building the deployment.
	DeploymentStatusBuilding = "building"
	// DeploymentStatusReady defines the status that means the deployment is built successfully.
	DeploymentStatusReady = "ready"
	// DeploymentStatusFailed defines the status that means the build attempt failed.
	DeploymentStatusFailed = "failed"
	// DeploymentStatusClosed defines the status that means the deployment is closed.
	DeploymentStatusClosed = "closed"
)

// Deployment is a model that represents a single deployment.
type Deployment struct {
	ID          uint64             `json:"id"`
	Status      string             `json:"status"`
	CreatedAt   time.Time          `json:"createdAt"`
	AutoRebuild bool               `json:"autoRebuild"`
	Branches    []DeploymentBranch `json:"branches"`
}

// DeploymentBranch is a model that contains a snapshot of the branch data used in the particular deployment.
type DeploymentBranch struct {
	ID   uint64 `json:"id"`
	Hash string `json:"hash"`
}

// FormAddDeployment represents a form of new deployment.
type FormAddDeployment struct {
	AutoRebuild bool     `json:"autoRebuild"`
	Branches    []uint64 `json:"branches"`
}
