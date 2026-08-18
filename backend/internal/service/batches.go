package service

import "warehouse/internal/store"

func (s *Service) ListLocations() ([]store.Location, error) {
	return s.Store.ListLocations()
}

func (s *Service) ListBatches(productID int64, q string) ([]store.Batch, error) {
	return s.Store.ListBatches(productID, q)
}
