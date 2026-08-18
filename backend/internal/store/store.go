package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var (
	ErrNotFound          = errors.New("record not found")
	ErrInsufficientStock = errors.New("insufficient stock")
)

type Store struct {
	db       *sql.DB
	operator string
}

type Product struct {
	ID            int64  `json:"id"`
	Name          string `json:"name"`
	Spec          string `json:"spec"`
	Unit          string `json:"unit"`
	Category      string `json:"category"`
	Stock         int64  `json:"stock"`
	AvgPriceCents int64  `json:"avgPriceCents"`
	Locations     string `json:"locations"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt"`
}

type Movement struct {
	ID           int64  `json:"id"`
	Type         string `json:"type"`
	ProductID    int64  `json:"productId"`
	ProductName  string `json:"productName"`
	Qty          int64  `json:"qty"`
	StockBefore  int64  `json:"stockBefore"`
	StockAfter   int64  `json:"stockAfter"`
	TargetQty    *int64 `json:"targetQty,omitempty"`
	Locations    string `json:"locations"`
	BatchRefs    string `json:"batchRefs"`
	Counterparty string `json:"counterparty"`
	Note         string `json:"note"`
	CreatedAt    string `json:"createdAt"`
}

type MovementParams struct {
	Type         string
	ProductID    int64
	ProductName  string
	Spec         string
	Qty          int64
	Counterparty string
	Note         string
	UnitPrice    int64
	LocationID   *int64
	LocationName string
}

type AuditEntry struct {
	ID       int64          `json:"id"`
	Ts       string         `json:"ts"`
	Operator string         `json:"operator"`
	Action   string         `json:"action"`
	Entity   string         `json:"entity"`
	EntityID int64          `json:"entityId,omitempty"`
	Detail   map[string]any `json:"detail"`
}

type Stats struct {
	SkuCount       int64 `json:"skuCount"`
	TotalUnits     int64 `json:"totalUnits"`
	InventoryValue int64 `json:"inventoryValue"`
	InToday        int64 `json:"inToday"`
	OutToday       int64 `json:"outToday"`
	InTotal        int64 `json:"inTotal"`
	OutTotal       int64 `json:"outTotal"`
	MovementCount  int64 `json:"movementCount"`
}

type Location struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	QtyLeft   int64  `json:"qtyLeft"`
	CreatedAt string `json:"createdAt"`
}

type Batch struct {
	ID             int64  `json:"id"`
	ProductID      int64  `json:"productId"`
	ProductName    string `json:"productName"`
	Qty            int64  `json:"qty"`
	QtyLeft        int64  `json:"qtyLeft"`
	UnitPriceCents int64  `json:"unitPriceCents"`
	Location       string `json:"location"`
	Supplier       string `json:"supplier"`
	Note           string `json:"note"`
	CreatedAt      string `json:"createdAt"`
}

func Open(path string, operator string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, operator: operator}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error     { return s.db.Close() }
func (s *Store) Operator() string { return s.operator }

func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS products (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  spec TEXT NOT NULL DEFAULT '',
  unit TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL DEFAULT '',
  stock INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS movements (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL CHECK (type IN ('in','out','adjust')),
  product_id INTEGER NOT NULL REFERENCES products(id),
  qty INTEGER NOT NULL,
  stock_before INTEGER NOT NULL,
  stock_after INTEGER NOT NULL,
  target_qty INTEGER,
  counterparty TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS audit_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts TEXT NOT NULL,
  operator TEXT NOT NULL,
  action TEXT NOT NULL,
  entity TEXT NOT NULL,
  entity_id INTEGER,
  detail TEXT NOT NULL DEFAULT '{}'
);
CREATE TABLE IF NOT EXISTS locations (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS batches (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  product_id INTEGER NOT NULL REFERENCES products(id),
  qty INTEGER NOT NULL,
  qty_left INTEGER NOT NULL,
  unit_price_cents INTEGER NOT NULL DEFAULT 0,
  location_id INTEGER REFERENCES locations(id),
  supplier TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS batch_ops (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  batch_id INTEGER NOT NULL REFERENCES batches(id),
  movement_id INTEGER NOT NULL REFERENCES movements(id),
  qty INTEGER NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_movements_product ON movements(product_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_audit_ts ON audit_log(ts DESC);
CREATE INDEX IF NOT EXISTS idx_batches_product ON batches(product_id, id);
CREATE INDEX IF NOT EXISTS idx_batch_ops_batch ON batch_ops(batch_id);
`
	_, err := s.db.Exec(strings.TrimSpace(schema))
	return err
}

func now() string { return time.Now().Format("2006-01-02 15:04:05") }

func (s *Store) log(tx *sql.Tx, action, entity string, entityID int64, detail map[string]any) error {
	b, err := json.Marshal(detail)
	if err != nil {
		return err
	}
	_, err = tx.Exec(
		`INSERT INTO audit_log(ts, operator, action, entity, entity_id, detail) VALUES (?,?,?,?,?,?)`,
		now(), s.operator, action, entity, entityID, string(b),
	)
	return err
}

func (s *Store) ListProducts(q string) ([]Product, error) {
	rows, err := s.db.Query(`
		SELECT p.id, p.name, p.spec, p.unit, p.category, p.created_at, p.updated_at,
		       COALESCE((SELECT SUM(b.qty_left) FROM batches b WHERE b.product_id = p.id), 0),
		       COALESCE((SELECT CAST(ROUND(1.0*SUM(b.qty_left*b.unit_price_cents)/NULLIF(SUM(b.qty_left),0)) AS INTEGER)
		                  FROM batches b WHERE b.product_id = p.id), 0),
		       COALESCE((SELECT GROUP_CONCAT(l.name || ' (' || loc.qty || ')')
		                  FROM (SELECT b.location_id, SUM(b.qty_left) AS qty
		                        FROM batches b
		                        WHERE b.product_id = p.id AND b.location_id IS NOT NULL
		                        GROUP BY b.location_id) loc
		                  LEFT JOIN locations l ON l.id = loc.location_id), '')
		FROM products p
		WHERE ? = '' OR name LIKE '%'||?||'%' OR category LIKE '%'||?||'%' OR spec LIKE '%'||?||'%'
		ORDER BY p.name, p.id`, q, q, q, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Spec, &p.Unit, &p.Category, &p.CreatedAt, &p.UpdatedAt,
			&p.Stock, &p.AvgPriceCents, &p.Locations); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) CreateProduct(name, spec, unit, category string) (Product, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Product{}, err
	}
	defer tx.Rollback()

	ts := now()
	res, err := tx.Exec(
		`INSERT INTO products(name, spec, unit, category, stock, created_at, updated_at) VALUES (?,?,?,?,0,?,?)`,
		name, spec, unit, category, ts, ts,
	)
	if err != nil {
		return Product{}, err
	}
	id, _ := res.LastInsertId()
	if err := s.log(tx, "product.create", "product", id, map[string]any{
		"name": name, "spec": spec, "unit": unit, "category": category,
	}); err != nil {
		return Product{}, err
	}
	if err := tx.Commit(); err != nil {
		return Product{}, err
	}
	return Product{ID: id, Name: name, Spec: spec, Unit: unit, Category: category, CreatedAt: ts, UpdatedAt: ts}, nil
}

func (s *Store) UpdateProduct(id int64, name, spec, unit, category string) (Product, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Product{}, err
	}
	defer tx.Rollback()

	var old Product
	err = tx.QueryRow(`SELECT id, name, spec, unit, category FROM products WHERE id = ?`, id).
		Scan(&old.ID, &old.Name, &old.Spec, &old.Unit, &old.Category)
	if err == sql.ErrNoRows {
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, err
	}

	ts := now()
	_, err = tx.Exec(`UPDATE products SET name=?, spec=?, unit=?, category=?, updated_at=? WHERE id=?`,
		name, spec, unit, category, ts, id)
	if err != nil {
		return Product{}, err
	}
	if err := s.log(tx, "product.update", "product", id, map[string]any{
		"before": old,
		"after":  Product{ID: id, Name: name, Spec: spec, Unit: unit, Category: category},
	}); err != nil {
		return Product{}, err
	}
	if err := tx.Commit(); err != nil {
		return Product{}, err
	}
	var p Product
	err = s.db.QueryRow(`SELECT id, name, spec, unit, category, stock, created_at, updated_at FROM products WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Spec, &p.Unit, &p.Category, &p.Stock, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (s *Store) AddMovement(in MovementParams) (Movement, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Movement{}, err
	}
	defer tx.Rollback()
	m, err := s.addMovementTx(tx, in)
	if err != nil {
		return Movement{}, err
	}
	if err := tx.Commit(); err != nil {
		return Movement{}, err
	}
	return m, nil
}

func (s *Store) addMovementTx(tx *sql.Tx, in MovementParams) (Movement, error) {
	if in.ProductID <= 0 {
		return Movement{}, errors.New("product is required")
	}
	if in.Qty < 0 || (in.Type != "adjust" && in.Qty == 0) {
		return Movement{}, errors.New("invalid quantity")
	}
	if in.Type == "in" && in.UnitPrice < 0 {
		return Movement{}, errors.New("invalid unit price")
	}
	if in.LocationName != "" {
		id, err := s.ensureLocationTx(tx, in.LocationName)
		if err != nil {
			return Movement{}, err
		}
		in.LocationID = &id
	}

	var (
		cur    int64
		name   string
		ts     = now()
		delta  int64
		after  int64
		tgt    *int64
		allocs []map[string]any
		base   int64
	)
	err := tx.QueryRow(`SELECT stock, name FROM products WHERE id = ?`, in.ProductID).Scan(&cur, &name)
	if err == sql.ErrNoRows {
		return Movement{}, ErrNotFound
	}
	if err != nil {
		return Movement{}, err
	}

	switch in.Type {
	case "in":
		delta, after = in.Qty, cur+in.Qty
	case "out":
		if in.Qty > cur {
			return Movement{}, ErrInsufficientStock
		}
		delta, after = -in.Qty, cur-in.Qty
	case "adjust":
		base = cur
		if in.LocationID != nil {
			if err := tx.QueryRow(
				`SELECT COALESCE(SUM(qty_left),0) FROM batches WHERE product_id = ? AND location_id = ?`,
				in.ProductID, *in.LocationID,
			).Scan(&base); err != nil {
				return Movement{}, err
			}
		}
		t := in.Qty
		tgt = &t
		delta = in.Qty - base
		after = cur + delta
	default:
		return Movement{}, errors.New("invalid movement type")
	}

	res, err := tx.Exec(
		`INSERT INTO movements(type, product_id, qty, stock_before, stock_after, target_qty, counterparty, note, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		in.Type, in.ProductID, delta, cur, after, tgt, in.Counterparty, in.Note, ts,
	)
	if err != nil {
		return Movement{}, err
	}
	mid, _ := res.LastInsertId()

	switch in.Type {
	case "in":
		bid, err := s.insertBatch(tx, in.ProductID, in.Qty, in.Qty, in.UnitPrice, in.LocationID, in.Counterparty, in.Note, ts)
		if err != nil {
			return Movement{}, err
		}
		if err := s.insertBatchOp(tx, bid, mid, in.Qty, ts); err != nil {
			return Movement{}, err
		}
	case "out":
		ops, err := s.consumeBatches(tx, in.ProductID, in.Qty, mid, ts, in.LocationID)
		if err != nil {
			return Movement{}, err
		}
		allocs = ops
	case "adjust":
		diff := delta
		if diff > 0 {
			var bid int64
			q := `SELECT id FROM batches WHERE product_id = ? AND qty_left > 0 ORDER BY id LIMIT 1`
			args := []any{in.ProductID}
			if in.LocationID != nil {
				q = `SELECT id FROM batches WHERE product_id = ? AND location_id = ? AND qty_left > 0 ORDER BY id LIMIT 1`
				args = append(args, *in.LocationID)
			}
			err := tx.QueryRow(q, args...).Scan(&bid)
			if err == sql.ErrNoRows {
				if bid, err = s.insertBatch(tx, in.ProductID, diff, diff, 0, in.LocationID, "", "盘点新增", ts); err != nil {
					return Movement{}, err
				}
			} else if err != nil {
				return Movement{}, err
			} else if _, err := tx.Exec(`UPDATE batches SET qty_left = qty_left + ? WHERE id = ?`, diff, bid); err != nil {
				return Movement{}, err
			}
			if err := s.insertBatchOp(tx, bid, mid, diff, ts); err != nil {
				return Movement{}, err
			}
			allocs = []map[string]any{{"batchId": bid, "qty": diff}}
		} else if diff < 0 {
			ops, err := s.consumeBatches(tx, in.ProductID, -diff, mid, ts, in.LocationID)
			if err != nil {
				return Movement{}, err
			}
			allocs = ops
		}
	}

	if _, err := tx.Exec(`UPDATE products SET stock=?, updated_at=? WHERE id=?`, after, ts, in.ProductID); err != nil {
		return Movement{}, err
	}

	detail := map[string]any{
		"type": in.Type, "product": name, "qty": in.Qty, "stockBefore": cur, "stockAfter": after,
		"counterparty": in.Counterparty, "note": in.Note,
	}
	if in.Type == "in" {
		detail["priceCents"] = in.UnitPrice
	}
	if in.Type == "adjust" {
		detail["from"] = base
	}
	if in.LocationID != nil {
		var loc string
		if err := tx.QueryRow(`SELECT name FROM locations WHERE id = ?`, *in.LocationID).Scan(&loc); err == nil {
			detail["location"] = loc
		}
	}
	if tgt != nil {
		detail["target"] = *tgt
	}
	if len(allocs) > 0 {
		detail["allocations"] = allocs
	}
	if err := s.log(tx, "stock."+in.Type, "movement", mid, detail); err != nil {
		return Movement{}, err
	}
	return Movement{
		ID: mid, Type: in.Type, ProductID: in.ProductID, ProductName: name, Qty: delta,
		StockBefore: cur, StockAfter: after, TargetQty: tgt, Counterparty: in.Counterparty,
		Note: in.Note, CreatedAt: ts,
	}, nil
}

// RowError identifies which input row failed inside a batch.
type RowError struct {
	Row int
	Err error
}

func (e *RowError) Error() string {
	return fmt.Sprintf("第%d行: %s", e.Row, e.Err.Error())
}

func (e *RowError) Unwrap() error { return e.Err }

// BatchMovements applies all movements in one transaction and returns the
// number of applied rows. Inbound rows may reference products that do not
// exist yet; they are created with default unit and category. Other row kinds
// require an existing product.
func (s *Store) BatchMovements(items []MovementParams) (int, error) {
	if len(items) == 0 {
		return 0, errors.New("empty batch")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	for i, in := range items {
		if in.ProductName == "" {
			return 0, &RowError{Row: i + 1, Err: errors.New("商品名称为空")}
		}
		var pid int64
		err := tx.QueryRow(
			`SELECT id FROM products WHERE name = ? AND spec = ? LIMIT 1`,
			in.ProductName, in.Spec,
		).Scan(&pid)
		if err == sql.ErrNoRows {
			if in.Type != "in" {
				return 0, &RowError{Row: i + 1, Err: ErrNotFound}
			}
			ts := now()
			res, err := tx.Exec(
				`INSERT INTO products(name, spec, unit, category, stock, created_at, updated_at) VALUES (?,?,?,?,0,?,?)`,
				in.ProductName, in.Spec, "件", "", ts, ts,
			)
			if err != nil {
				return 0, err
			}
			pid, _ = res.LastInsertId()
			if err := s.log(tx, "product.create", "product", pid, map[string]any{
				"name": in.ProductName, "spec": in.Spec, "unit": "件", "category": "",
			}); err != nil {
				return 0, err
			}
		} else if err != nil {
			return 0, err
		}
		in.ProductID = pid
		if _, err := s.addMovementTx(tx, in); err != nil {
			return 0, &RowError{Row: i + 1, Err: err}
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(items), nil
}

func (s *Store) insertBatch(tx *sql.Tx, productID, qty, qtyLeft, unitPrice int64, locationID *int64, supplier, note, ts string) (int64, error) {
	res, err := tx.Exec(
		`INSERT INTO batches(product_id, qty, qty_left, unit_price_cents, location_id, supplier, note, created_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
		productID, qty, qtyLeft, unitPrice, locationID, supplier, note, ts,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) insertBatchOp(tx *sql.Tx, batchID, movementID, qty int64, ts string) error {
	_, err := tx.Exec(`INSERT INTO batch_ops(batch_id, movement_id, qty, created_at) VALUES (?,?,?,?)`,
		batchID, movementID, qty, ts)
	return err
}

// consumeBatches deducts qty from the product's batches FIFO (oldest first)
// and returns per-batch allocations. Returns ErrInsufficientStock when total
// remaining stock is insufficient.
func (s *Store) consumeBatches(tx *sql.Tx, productID, qty, movementID int64, ts string, locationID *int64) ([]map[string]any, error) {
	where := `SELECT id, qty_left FROM batches WHERE product_id = ? AND qty_left > 0`
	args := []any{productID}
	if locationID != nil {
		where += ` AND location_id = ?`
		args = append(args, *locationID)
	}
	where += ` ORDER BY id`
	rows, err := tx.Query(where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allocs []map[string]any
	remaining := qty
	for rows.Next() {
		if remaining == 0 {
			break
		}
		var bid, left int64
		if err := rows.Scan(&bid, &left); err != nil {
			return nil, err
		}
		take := remaining
		if left < take {
			take = left
		}
		if _, err := tx.Exec(`UPDATE batches SET qty_left = qty_left - ? WHERE id = ?`, take, bid); err != nil {
			return nil, err
		}
		if err := s.insertBatchOp(tx, bid, movementID, -take, ts); err != nil {
			return nil, err
		}
		allocs = append(allocs, map[string]any{"batchId": bid, "qty": -take})
		remaining -= take
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if remaining > 0 {
		return nil, ErrInsufficientStock
	}
	return allocs, nil
}

func (s *Store) ListMovements(mtype string, productID int64, q string, limit int) ([]Movement, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	where := "WHERE 1=1"
	args := []any{}
	if mtype != "" {
		where += " AND m.type = ?"
		args = append(args, mtype)
	}
	if productID > 0 {
		where += " AND m.product_id = ?"
		args = append(args, productID)
	}
	if q != "" {
		where += " AND (p.name LIKE '%'||?||'%' OR m.counterparty LIKE '%'||?||'%' OR m.note LIKE '%'||?||'%')"
		args = append(args, q, q, q)
	}
	args = append(args, limit)

	rows, err := s.db.Query(`
		SELECT m.id, m.type, m.product_id, p.name, m.qty, m.stock_before, m.stock_after,
		       m.target_qty,
		       COALESCE((SELECT GROUP_CONCAT(DISTINCT l.name)
		                 FROM batch_ops bo
		                 JOIN batches b ON b.id = bo.batch_id
		                 LEFT JOIN locations l ON l.id = b.location_id
		                 WHERE bo.movement_id = m.id AND l.name IS NOT NULL), ''),
		       COALESCE((SELECT GROUP_CONCAT(bo.batch_id || 'x' || -bo.qty)
		                 FROM batch_ops bo
		                 WHERE bo.movement_id = m.id AND bo.qty < 0), ''),
		       m.counterparty, m.note, m.created_at
		FROM movements m JOIN products p ON p.id = m.product_id
		`+where+`
		ORDER BY m.id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Movement
	for rows.Next() {
		var m Movement
		var tgt sql.NullInt64
		if err := rows.Scan(&m.ID, &m.Type, &m.ProductID, &m.ProductName, &m.Qty, &m.StockBefore,
			&m.StockAfter, &tgt, &m.Locations, &m.BatchRefs, &m.Counterparty, &m.Note, &m.CreatedAt); err != nil {
			return nil, err
		}
		if tgt.Valid {
			v := tgt.Int64
			m.TargetQty = &v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) ListAudit(action, q string, limit int) ([]AuditEntry, error) {
	if limit <= 0 || limit > 1000 {
		limit = 300
	}
	where := "WHERE 1=1"
	args := []any{}
	if action != "" {
		where += " AND action = ?"
		args = append(args, action)
	}
	if q != "" {
		where += " AND (detail LIKE '%'||?||'%' OR entity LIKE '%'||?||'%')"
		args = append(args, q, q)
	}
	args = append(args, limit)

	rows, err := s.db.Query(`
		SELECT id, ts, operator, action, entity, COALESCE(entity_id,0), detail
		FROM audit_log `+where+`
		ORDER BY id DESC LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AuditEntry
	for rows.Next() {
		var a AuditEntry
		var detail string
		if err := rows.Scan(&a.ID, &a.Ts, &a.Operator, &a.Action, &a.Entity, &a.EntityID, &detail); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(detail), &a.Detail); err != nil {
			a.Detail = map[string]any{"raw": detail}
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) EnsureLocation(name string) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	id, err := s.ensureLocationTx(tx, name)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) ensureLocationTx(tx *sql.Tx, name string) (int64, error) {
	ts := now()
	if _, err := tx.Exec(
		`INSERT INTO locations(name, created_at) VALUES (?,?) ON CONFLICT(name) DO NOTHING`,
		name, ts,
	); err != nil {
		return 0, err
	}
	var id int64
	if err := tx.QueryRow(`SELECT id FROM locations WHERE name = ?`, name).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Store) ListLocations() ([]Location, error) {
	rows, err := s.db.Query(`
		SELECT l.id, l.name, l.created_at,
		       COALESCE((SELECT SUM(b.qty_left) FROM batches b WHERE b.location_id = l.id), 0)
		FROM locations l
		ORDER BY l.name, l.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Location
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.Name, &l.CreatedAt, &l.QtyLeft); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) ListBatches(productID int64, q string) ([]Batch, error) {
	where := "WHERE 1=1"
	args := []any{}
	if productID > 0 {
		where += " AND b.product_id = ?"
		args = append(args, productID)
	}
	if q != "" {
		where += " AND (p.name LIKE '%'||?||'%' OR l.name LIKE '%'||?||'%' OR b.supplier LIKE '%'||?||'%')"
		args = append(args, q, q, q)
	}

	rows, err := s.db.Query(`
		SELECT b.id, b.product_id, p.name, b.qty, b.qty_left, b.unit_price_cents,
		       COALESCE(l.name,''), b.supplier, b.note, b.created_at
		FROM batches b
		JOIN products p ON p.id = b.product_id
		LEFT JOIN locations l ON l.id = b.location_id
		`+where+`
		ORDER BY b.id DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Batch
	for rows.Next() {
		var b Batch
		if err := rows.Scan(&b.ID, &b.ProductID, &b.ProductName, &b.Qty, &b.QtyLeft,
			&b.UnitPriceCents, &b.Location, &b.Supplier, &b.Note, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *Store) Stats() (Stats, error) {
	var st Stats
	if err := s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(stock),0) FROM products`).Scan(&st.SkuCount, &st.TotalUnits); err != nil {
		return st, err
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(qty_left*unit_price_cents),0) FROM batches`).Scan(&st.InventoryValue); err != nil {
		return st, err
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM movements WHERE type='in' AND date(created_at)=date('now','localtime')`).Scan(&st.InToday); err != nil {
		return st, err
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(-qty),0) FROM movements WHERE type='out' AND date(created_at)=date('now','localtime')`).Scan(&st.OutToday); err != nil {
		return st, err
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(qty),0) FROM movements WHERE type='in'`).Scan(&st.InTotal); err != nil {
		return st, err
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(-qty),0) FROM movements WHERE type='out'`).Scan(&st.OutTotal); err != nil {
		return st, err
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM movements`).Scan(&st.MovementCount); err != nil {
		return st, err
	}
	return st, nil
}
