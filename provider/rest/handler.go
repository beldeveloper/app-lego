package rest

import (
	"encoding/json"
	"fmt"
	"github.com/beldeveloper/app-lego/controller"
	"github.com/beldeveloper/app-lego/model"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strconv"
)

// NewHandler creates a new instance of the REST API handler.
func NewHandler(c controller.Service) Handler {
	return Handler{c: c}
}

// Handler handles the RESP API requests.
type Handler struct {
	c controller.Service
}

// Repositories returns the list of repositories.
func (h Handler) Repositories(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	res, err := h.c.Repositories(r.Context())
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, res)
}

// AddRepository adds new repository and puts int to pending download status.
func (h Handler) AddRepository(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var f model.FormAddRepository
	err := json.NewDecoder(r.Body).Decode(&f)
	if err != nil {
		apiError(w, err)
		return
	}
	res, err := h.c.AddRepository(r.Context(), f)
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, res)
}

// Branches returns the list of branches.
func (h Handler) Branches(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	res, err := h.c.Branches(r.Context())
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, res)
}

// Deployments returns the list of deployments.
func (h Handler) Deployments(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	res, err := h.c.Deployments(r.Context())
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, res)
}

// AddDeployment adds and enqueues new deployment.
func (h Handler) AddDeployment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var f model.FormAddDeployment
	err := json.NewDecoder(r.Body).Decode(&f)
	if err != nil {
		apiError(w, err)
		return
	}
	res, err := h.c.AddDeployment(r.Context(), f)
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, res)
}

// RebuildDeployment enqueues the existing deployment for rebuilding.
func (h Handler) RebuildDeployment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		apiError(w, fmt.Errorf("%w: invalid deployment id: %v", model.ErrBadInput, err))
		return
	}
	res, err := h.c.RebuildDeployment(r.Context(), uint64(id))
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, res)
}

// CloseDeployment enqueues the existing deployment for closing.
func (h Handler) CloseDeployment(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	id, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		apiError(w, fmt.Errorf("%w: invalid deployment id: %v", model.ErrBadInput, err))
		return
	}
	err = h.c.CloseDeployment(r.Context(), uint64(id))
	if err != nil {
		apiError(w, err)
		return
	}
	apiSuccess(w, nil)
}
