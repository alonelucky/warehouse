// Package service holds the business layer. Controllers call its methods
// and map returned errors to HTTP responses.
package service

import "warehouse/internal/store"

type Service struct {
	Store *store.Store
}

func New(st *store.Store) *Service {
	return &Service{Store: st}
}
