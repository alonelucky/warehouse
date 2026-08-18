// Package controller holds HTTP handlers. Business logic lives in Service.
package controller

import (
	"encoding/json"
	"net/http"

	"warehouse/internal/service"
)

type Controller struct {
	SVC *service.Service
}

func New(svc *service.Service) *Controller {
	return &Controller{SVC: svc}
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
