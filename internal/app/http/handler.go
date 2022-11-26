package http

import (
	"encoding/json"
	"fmt"
	"github.com/beldeveloper/app-lego/internal/app"
	"github.com/beldeveloper/app-lego/internal/app/errtype"
	"github.com/beldeveloper/go-errors-context"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strconv"
)

// NewHandler creates a new instance of the REST API handler.
func NewHandler(
	repoSvc app.RepositorySvc,
	branchSvc app.BranchSvc,
	deploySvc app.DeploymentSvc,
	accessKey app.ApiAccessKey,
) Handler {
	return Handler{
		repoSvc:   repoSvc,
		branchSvc: branchSvc,
		deploySvc: deploySvc,
		accessKey: string(accessKey),
	}
}

// Handler handles the RESP API requests.
type Handler struct {
	repoSvc   app.RepositorySvc
	branchSvc app.BranchSvc
	deploySvc app.DeploymentSvc
	accessKey string
}

// Repositories returns the list of repositories.
func (h Handler) Repositories(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := h.validateKey(r)
	if err != nil {
		apiError(w, err)
		return
	}
	res, err := h.repoSvc.List(r.Context())
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, res)
}

// AddRepository adds new repository and puts int to pending download status.
func (h Handler) AddRepository(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := h.validateKey(r)
	if err != nil {
		apiError(w, err)
		return
	}
	var f app.FormAddRepository
	err = json.NewDecoder(r.Body).Decode(&f)
	if err != nil {
		apiError(w, err)
		return
	}
	res, err := h.repoSvc.Add(r.Context(), f)
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, res)
}

// Branches returns the list of branches.
func (h Handler) Branches(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := h.validateKey(r)
	if err != nil {
		apiError(w, err)
		return
	}
	res, err := h.branchSvc.List(r.Context())
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, res)
}

// Deployments returns non-closed deployments.
func (h Handler) Deployments(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := h.validateKey(r)
	if err != nil {
		apiError(w, err)
		return
	}
	res, err := h.deploySvc.List(r.Context())
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, res)
}

// AddDeployment adds and enqueues new deployment.
func (h Handler) AddDeployment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := h.validateKey(r)
	if err != nil {
		apiError(w, err)
		return
	}
	var f app.FormAddDeployment
	err = json.NewDecoder(r.Body).Decode(&f)
	if err != nil {
		apiError(w, err)
		return
	}
	res, err := h.deploySvc.Add(r.Context(), f)
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, res)
}

// RebuildDeployment enqueues the existing deployment for rebuilding.
func (h Handler) RebuildDeployment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := h.validateKey(r)
	if err != nil {
		apiError(w, err)
		return
	}
	id, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		apiError(w, fmt.Errorf("%w: invalid deployment id: %v", errtype.ErrBadInput, err))
		return
	}
	var f app.FormReDeployment
	err = json.NewDecoder(r.Body).Decode(&f)
	if err != nil {
		apiError(w, err)
		return
	}
	f.ID = uint64(id)
	res, err := h.deploySvc.Rebuild(r.Context(), f)
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, res)
}

// CloseDeployment enqueues the existing deployment for closing.
func (h Handler) CloseDeployment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := h.validateKey(r)
	if err != nil {
		apiError(w, err)
		return
	}
	id, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		apiError(w, fmt.Errorf("%w: invalid deployment id: %v", errtype.ErrBadInput, err))
		return
	}
	err = h.deploySvc.Close(r.Context(), uint64(id))
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, nil)
}

func (h Handler) validateKey(r *http.Request) error {
	if r.URL.Query().Get("accessKey") != h.accessKey {
		return errors.WrapContext(errtype.ErrUnauthorized, errors.Context{})
	}
	return nil
}
