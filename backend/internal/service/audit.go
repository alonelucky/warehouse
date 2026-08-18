package service

import "warehouse/internal/store"

func (s *Service) ListAudit(action, q string, limit int) ([]store.AuditEntry, error) {
	return s.Store.ListAudit(action, q, limit)
}
