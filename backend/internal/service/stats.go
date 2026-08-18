package service

import "warehouse/internal/store"

func (s *Service) Stats() (store.Stats, error) {
	return s.Store.Stats()
}

func (s *Service) Operator() string {
	return s.Store.Operator()
}
