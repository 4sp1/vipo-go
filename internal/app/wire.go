package app

import (
	"encoding/json"
	"net/http"

	"github.com/4sp1/vipo-go/internal/app/apperr"
	oapi "github.com/4sp1/vipo-go/internal/port/oapi"
	"github.com/4sp1/vipo-go/internal/svc"
)

func NewHandler(s *svc.Service) http.Handler {
	strictHandler := oapi.NewStrictHandlerWithOptions(s, nil, oapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			code, msg, _ := apperr.Wire(err)
			wireCode := code.String()
			if code == apperr.Code(-1) {
				wireCode = apperr.CodeInvalidRequest.String()
			}
			json.NewEncoder(w).Encode(oapi.Error{Code: &wireCode, Message: msg})
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			code, msg, status := apperr.Wire(err)
			if status == 500 {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			wireCode := code.String()
			json.NewEncoder(w).Encode(oapi.Error{Code: &wireCode, Message: msg})
		},
	})

	return oapi.Handler(strictHandler)
}
