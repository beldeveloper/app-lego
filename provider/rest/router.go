package rest

import (
	"github.com/beldeveloper/app-lego/controller"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

// CreateRouter creates and configures a new instance of the router.
func CreateRouter(c controller.Service) *httprouter.Router {
	h := NewHandler(c)
	r := httprouter.New()

	r.GET("/repositories", h.Repositories)
	r.POST("/repositories", h.AddRepository)
	r.GET("/branches", h.Branches)
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
