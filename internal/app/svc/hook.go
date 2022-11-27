package svc

import (
	"context"
	"github.com/beldeveloper/app-lego/pkg"
	"github.com/beldeveloper/app-lego/rpc/hook"
	"github.com/beldeveloper/go-errors-context"
)

// NewHook creates a new instance of the hook client.
func NewHook(client hook.HookClient) pkg.HookSvc {
	return Hook{client: client}
}

// Hook implements a hook client.
type Hook struct {
	client hook.HookClient
}

// BuildBranch calls hook handler in order to build a specific branch.
func (s Hook) BuildBranch(ctx context.Context, req pkg.HookBuildBranchReq) (pkg.HookBuildBranchResp, error) {
	var res pkg.HookBuildBranchResp
	rpcRes, err := s.client.BuildBranch(ctx, &hook.BuildBranchReq{
		Repo: &hook.Repo{
			Id:    req.Repo.ID,
			Type:  req.Repo.Type,
			Alias: req.Repo.Alias,
		},
		Branch: &hook.Branch{
			Id:     req.Branch.ID,
			RepoId: req.Branch.RepoID,
			Type:   req.Branch.Type,
			Name:   req.Branch.Name,
			Hash:   req.Branch.Hash,
		},
	})
	if err != nil {
		return res, errors.WrapContext(err, errors.Context{Path: "svc.Hook.BuildBranch"})
	}
	res.Status = rpcRes.Status
	if rpcRes.ErrorMsg != "" {
		res.ErrorMsg = &rpcRes.ErrorMsg
	}
	return res, nil
}

// Deploy calls hook handler in order to perform new deployment.
func (s Hook) Deploy(ctx context.Context, req pkg.HookDeployReq) (pkg.HookDeployResp, error) {
	var res pkg.HookDeployResp
	rpcReq := &hook.DeployReq{
		Repos:       make([]*hook.Repo, len(req.Repos)),
		Deployments: make([]*hook.Deployment, len(req.Deployments)),
	}
	for i, r := range req.Repos {
		rpcReq.Repos[i] = &hook.Repo{
			Id:    r.ID,
			Type:  r.Type,
			Alias: r.Alias,
		}
	}
	for i, d := range req.Deployments {
		rpcDep := &hook.Deployment{
			Id:       d.ID,
			Updated:  d.Updated,
			Branches: make(map[string]*hook.Branch, len(d.Branches)),
		}
		for k, b := range d.Branches {
			rpcDep.Branches[k] = &hook.Branch{
				Id:     b.ID,
				RepoId: b.RepoID,
				Type:   b.Type,
				Name:   b.Name,
				Hash:   b.Hash,
			}
		}
		rpcReq.Deployments[i] = rpcDep
	}
	rpcRes, err := s.client.Deploy(ctx, rpcReq)
	if err != nil {
		return res, errors.WrapContext(err, errors.Context{Path: "svc.Hook.Deploy"})
	}
	res.Statuses = make(map[uint64]pkg.HookDeployStatus, len(rpcRes.Statuses))
	for k, v := range rpcRes.Statuses {
		status := pkg.HookDeployStatus{Status: v.Status}
		if v.ErrorMsg != "" {
			status.ErrorMsg = &v.ErrorMsg
		}
		res.Statuses[k] = status
	}
	return res, nil
}

// CleanBranches calls hook handler in order to clean deleted branches.
func (s Hook) CleanBranches(ctx context.Context, ids []uint64) error {
	_, err := s.client.CleanBranches(ctx, &hook.CleanBranchesReq{Ids: ids})
	return errors.WrapContext(err, errors.Context{Path: "svc.Hook.CleanBranches"})
}
