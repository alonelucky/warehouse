package controller

import (
	"net/http"

	"warehouse/pkg/web"
)

func (c *Controller) Stats(w http.ResponseWriter, r *http.Request) {
	st, err := c.SVC.Stats()
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, st)
}

func (c *Controller) Meta(w http.ResponseWriter, r *http.Request) {
	web.SendJson(w, map[string]any{"operator": c.SVC.Operator()})
}
