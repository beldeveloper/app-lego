package rest

import (
	"encoding/json"
	"errors"
	"github.com/beldeveloper/app-lego/model"
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
	case errors.Is(err, model.ErrNotFound):
		code = http.StatusNotFound
	case errors.Is(err, model.ErrBadInput):
		code = http.StatusBadRequest
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
