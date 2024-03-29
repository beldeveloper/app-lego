package http

import (
	"encoding/json"
	"github.com/beldeveloper/app-lego/internal/app/errtype"
	"github.com/beldeveloper/go-errors-context"
	"log"
	"net/http"
)

// SetDefaultHeaders sets the basic set of headers to the response.
func SetDefaultHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Headers", "Accept,Authorization,Accept-Language,Content-Type,Content-Language")
}

func apiError(w http.ResponseWriter, err error) {
	SetDefaultHeaders(w)
	code := http.StatusInternalServerError
	switch true {
	case errors.Is(err, errtype.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, errtype.ErrBadInput):
		code = http.StatusBadRequest
	case errors.Is(err, errtype.ErrUnauthorized):
		code = http.StatusUnauthorized
	default:
		log.Println(err)
	}
	w.WriteHeader(code)
}

func apiSuccess(w http.ResponseWriter, data interface{}) {
	SetDefaultHeaders(w)
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Println(err)
	}
}
