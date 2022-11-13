package pkg

import "context"

// HookRepo contains repository data for passing into hook handler.
type HookRepo struct {
	ID    uint64
	Type  string
	Alias string
}

// HookBranch contains branch data for passing into hook handler.
type HookBranch struct {
	ID     uint64
	RepoID uint64
	Type   string
	Name   string
	Hash   string
}

// HookDeployment contains deployment data for passing into hook handler.
type HookDeployment struct {
	ID       uint64
	Branches map[string]HookBranch
	Updated  bool
}

// HookBuildBranchReq contains request data for calling branch build in the hook handler.
type HookBuildBranchReq struct {
	Repo   HookRepo
	Branch HookBranch
}

// HookBuildBranchResp contains response data from the hook handler.
type HookBuildBranchResp struct {
	Status string
}

// HookDeployReq contains request data for calling deploy in the hook handler.
type HookDeployReq struct {
	Repos       []HookRepo
	Deployments []HookDeployment
}

// HookDeployResp contains response data from the hook handler.
type HookDeployResp struct {
	Statuses map[uint64]string
}

// HookSvc describes the interactions with the hook handler.
type HookSvc interface {
	BuildBranch(ctx context.Context, req HookBuildBranchReq) (HookBuildBranchResp, error)
	Deploy(ctx context.Context, req HookDeployReq) (HookDeployResp, error)
	CleanBranches(ctx context.Context, ids []uint64) error
}
