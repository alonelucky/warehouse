package service

import (
	"errors"
	"fmt"
	"strings"

	"warehouse/internal/store"
)

type MovementInput struct {
	Type           string `json:"type"`
	ProductID      int64  `json:"productId"`
	Qty            int64  `json:"qty"`
	Counterparty   string `json:"counterparty"`
	Note           string `json:"note"`
	UnitPriceCents int64  `json:"unitPriceCents"`
	Location       string `json:"location"`
}

func (s *Service) ListMovements(mtype string, productID int64, q string, limit int) ([]store.Movement, error) {
	return s.Store.ListMovements(mtype, productID, q, limit)
}

func (s *Service) AddMovement(in MovementInput) (store.Movement, error) {
	in.Type = strings.ToLower(strings.TrimSpace(in.Type))
	in.Counterparty = strings.TrimSpace(in.Counterparty)
	in.Note = strings.TrimSpace(in.Note)
	if in.ProductID <= 0 {
		return store.Movement{}, &Error{Status: 400, Msg: "请选择商品"}
	}
	if in.Qty < 0 || (in.Type != "adjust" && in.Qty == 0) {
		return store.Movement{}, &Error{Status: 400, Msg: "数量无效"}
	}
	if in.Type != "in" && in.Type != "out" && in.Type != "adjust" {
		return store.Movement{}, &Error{Status: 400, Msg: "流水类型无效"}
	}
	if in.Type == "in" && in.UnitPriceCents < 0 {
		return store.Movement{}, &Error{Status: 400, Msg: "单价无效"}
	}

	m, err := s.Store.AddMovement(store.MovementParams{
		Type:         in.Type,
		ProductID:    in.ProductID,
		Qty:          in.Qty,
		Counterparty: in.Counterparty,
		Note:         in.Note,
		UnitPrice:    in.UnitPriceCents,
		LocationName: in.Location,
	})
	if errors.Is(err, store.ErrNotFound) {
		return store.Movement{}, &Error{Status: 404, Msg: "商品不存在"}
	}
	if errors.Is(err, store.ErrInsufficientStock) {
		if in.Type == "out" && in.Location != "" {
			return store.Movement{}, &Error{Status: 400, Msg: "该货位库存不足"}
		}
		return store.Movement{}, &Error{Status: 400, Msg: "库存不足"}
	}
	return m, err
}

type BatchItem struct {
	ProductName    string `json:"productName"`
	Spec           string `json:"spec"`
	Type           string `json:"type"`
	Qty            int64  `json:"qty"`
	UnitPriceCents int64  `json:"unitPriceCents"`
	Location       string `json:"location"`
	Counterparty   string `json:"counterparty"`
	Note           string `json:"note"`
}

// BatchMovements validates every row up front, then applies all rows in one
// transaction. Failed rows abort the whole batch and carry the row number.
func (s *Service) BatchMovements(items []BatchItem) (int, error) {
	if len(items) == 0 {
		return 0, &Error{Status: 400, Msg: "没有可提交的数据"}
	}
	params := make([]store.MovementParams, 0, len(items))
	for i, it := range items {
		it.Type = strings.ToLower(strings.TrimSpace(it.Type))
		it.ProductName = strings.TrimSpace(it.ProductName)
		it.Counterparty = strings.TrimSpace(it.Counterparty)
		it.Note = strings.TrimSpace(it.Note)
		it.Location = strings.TrimSpace(it.Location)
		it.Spec = strings.TrimSpace(it.Spec)
		if it.ProductName == "" {
			return 0, &Error{Status: 400, Msg: fmt.Sprintf("第%d行: 商品名称为空", i+1)}
		}
		if it.Type != "in" && it.Type != "out" && it.Type != "adjust" {
			return 0, &Error{Status: 400, Msg: fmt.Sprintf("第%d行: 类型无效", i+1)}
		}
		if it.Qty < 0 || (it.Type != "adjust" && it.Qty == 0) {
			return 0, &Error{Status: 400, Msg: fmt.Sprintf("第%d行: 数量无效", i+1)}
		}
		if it.UnitPriceCents < 0 {
			return 0, &Error{Status: 400, Msg: fmt.Sprintf("第%d行: 单价无效", i+1)}
		}
		params = append(params, store.MovementParams{
			Type:         it.Type,
			ProductName:  it.ProductName,
			Spec:         it.Spec,
			Qty:          it.Qty,
			Counterparty: it.Counterparty,
			Note:         it.Note,
			UnitPrice:    it.UnitPriceCents,
			LocationName: it.Location,
		})
	}

	n, err := s.Store.BatchMovements(params)
	if err != nil {
		var re *store.RowError
		if errors.As(err, &re) {
			msg := re.Err.Error()
			switch {
			case errors.Is(re.Err, store.ErrNotFound):
				msg = "商品不存在"
			case errors.Is(re.Err, store.ErrInsufficientStock):
				msg = "库存不足"
			}
			return 0, &Error{Status: 400, Msg: fmt.Sprintf("第%d行: %s", re.Row, msg)}
		}
		return 0, err
	}
	return n, nil
}
