package controller

import (
	"net/http"
	"strconv"

	"warehouse/internal/service"
	"warehouse/pkg/web"
)

func (c *Controller) ListMovements(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	pid, _ := strconv.ParseInt(q.Get("productId"), 10, 64)
	limit, _ := strconv.Atoi(q.Get("limit"))
	items, err := c.SVC.ListMovements(q.Get("type"), pid, q.Get("q"), limit)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, items)
}

func (c *Controller) AddMovement(w http.ResponseWriter, r *http.Request) {
	var in service.MovementInput
	if err := decodeJSON(r, &in); err != nil {
		web.SendError(w, web.CodeBadRequest, "无效参数")
		return
	}
	m, err := c.SVC.AddMovement(in)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, m)
}

func (c *Controller) AddMovementBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Items []service.BatchItem `json:"items"`
	}
	if err := decodeJSON(r, &req); err != nil {
		web.SendError(w, web.CodeBadRequest, "无效参数")
		return
	}
	n, err := c.SVC.BatchMovements(req.Items)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, map[string]any{"count": n})
}
