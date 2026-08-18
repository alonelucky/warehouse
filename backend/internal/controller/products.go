package controller

import (
	"net/http"
	"strconv"

	"warehouse/internal/service"
	"warehouse/pkg/web"
)

func (c *Controller) ListProducts(w http.ResponseWriter, r *http.Request) {
	items, err := c.SVC.ListProducts(r.URL.Query().Get("q"))
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, items)
}

func (c *Controller) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var in service.ProductInput
	if err := decodeJSON(r, &in); err != nil {
		web.SendError(w, web.CodeBadRequest, "无效参数")
		return
	}
	p, err := c.SVC.CreateProduct(in)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, p)
}

func (c *Controller) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		web.SendError(w, web.CodeBadRequest, "无效 id")
		return
	}
	var in service.ProductInput
	if err := decodeJSON(r, &in); err != nil {
		web.SendError(w, web.CodeBadRequest, "无效参数")
		return
	}
	p, err := c.SVC.UpdateProduct(id, in)
	if err != nil {
		sendErr(w, err)
		return
	}
	web.SendJson(w, p)
}
