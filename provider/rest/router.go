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

	r.POST("/repositories", h.AddRepository)

	r.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetDefaultHeaders(w)
		h := w.Header()
		h.Set("Access-Control-Allow-Methods", h.Get("Allow"))
		w.WriteHeader(http.StatusNoContent)
	})

	return r
}
