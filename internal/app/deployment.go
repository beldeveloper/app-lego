package app

import (
	"context"
	"time"
)

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

// DeploymentSvc describes the deployment service.
type DeploymentSvc interface {
	List(context.Context) ([]Deployment, error)
	Add(context.Context, FormAddDeployment) (Deployment, error)
	Rebuild(context.Context, uint64) (Deployment, error)
	RebuildWithBranch(ctx context.Context, b Branch) error
	Close(context.Context, uint64) error
	WatchJob(ctx context.Context) error
}

// DeploymentRepo describes interactions with the deployment DB.
type DeploymentRepo interface {
	FindAll(ctx context.Context) ([]Deployment, error)
	FindForAutoRebuild(ctx context.Context, b Branch) ([]Deployment, error)
	FindByID(ctx context.Context, id uint64) (Deployment, error)
	Add(ctx context.Context, d Deployment) (Deployment, error)
	Update(ctx context.Context, d Deployment) (Deployment, error)
}
