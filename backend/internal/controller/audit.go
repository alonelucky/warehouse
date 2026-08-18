package controller

import (
	"net/http"
	"strconv"

	"warehouse/pkg/web"
)

func (c *Controller) ListAudit(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	items, err := c.SVC.ListAudit(q.Get("action"), q.Get("q"), limit)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, items)
}
