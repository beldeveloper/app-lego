package http

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
)

// NewRouter creates and configures a new instance of the router.
func NewRouter(h Handler) *httprouter.Router {
	r := httprouter.New()

	r.GET("/repositories", h.Repositories)
	r.POST("/repositories", h.AddRepository)
	r.GET("/branches", h.Branches)
	r.POST("/branch/:id", h.RebuildBranch)
	r.GET("/deployments", h.Deployments)
	r.POST("/deployments", h.AddDeployment)
	r.POST("/deployment/:id", h.RebuildDeployment)
	r.DELETE("/deployment/:id", h.CloseDeployment)

	r.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetDefaultHeaders(w)
		h := w.Header()
		h.Set("Access-Control-Allow-Methods", h.Get("Allow"))
		w.WriteHeader(http.StatusNoContent)
	})

	return r
}
