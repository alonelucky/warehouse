package controller

import (
	"net/http"
	"strconv"

	"warehouse/pkg/web"
)

func (c *Controller) ListLocations(w http.ResponseWriter, r *http.Request) {
	items, err := c.SVC.ListLocations()
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, items)
}

func (c *Controller) ListBatches(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pid, _ := strconv.ParseInt(q.Get("productId"), 10, 64)
	items, err := c.SVC.ListBatches(pid, q.Get("q"))
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, items)
}
