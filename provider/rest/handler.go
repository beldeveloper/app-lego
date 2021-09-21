package rest

import (
	"encoding/json"
	"github.com/beldeveloper/app-lego/controller"
	"github.com/beldeveloper/app-lego/model"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

// NewHandler creates a new instance of the REST API handler.
func NewHandler(c controller.Service) Handler {
	return Handler{c: c}
}

// Handler handles the RESP API requests.
type Handler struct {
	c controller.Service
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
