package service

import (
	"strings"

	"warehouse/internal/store"
)

type ProductInput struct {
	Name     string `json:"name"`
	Spec     string `json:"spec"`
	Unit     string `json:"unit"`
	Category string `json:"category"`
}

func (s *Service) ListProducts(q string) ([]store.Product, error) {
	return s.Store.ListProducts(q)
}

func (s *Service) CreateProduct(in ProductInput) (store.Product, error) {
	in = cleanProduct(in)
	if in.Name == "" {
		return store.Product{}, &Error{Status: 400, Msg: "商品名称必填"}
	}
	return s.Store.CreateProduct(in.Name, in.Spec, in.Unit, in.Category)
}

func (s *Service) UpdateProduct(id int64, in ProductInput) (store.Product, error) {
	in = cleanProduct(in)
	if in.Name == "" {
		return store.Product{}, &Error{Status: 400, Msg: "商品名称必填"}
	}
	p, err := s.Store.UpdateProduct(id, in.Name, in.Spec, in.Unit, in.Category)
	if err == store.ErrNotFound {
		return store.Product{}, &Error{Status: 404, Msg: "商品不存在"}
	}
	return p, err
}

func cleanProduct(in ProductInput) ProductInput {
	in.Name = strings.TrimSpace(in.Name)
	in.Spec = strings.TrimSpace(in.Spec)
	in.Unit = strings.TrimSpace(in.Unit)
	in.Category = strings.TrimSpace(in.Category)
	if in.Unit == "" {
		in.Unit = "件"
	}
	return in
}
