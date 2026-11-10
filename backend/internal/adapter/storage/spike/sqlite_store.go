package spike

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite WAL bekerja optimal dengan satu writer dan banyak reader; untuk single-process local DB cukup pool kecil.
	db.SetMaxOpenConns(4)

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA temp_store=MEMORY;",
		"PRAGMA cache_size=-16000;",
		"PRAGMA foreign_keys=ON;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("exec %s: %w", p, err)
		}
	}

	schema := `
	CREATE TABLE IF NOT EXISTS medicines (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		name_lower TEXT NOT NULL,
		barcode TEXT NOT NULL,
		sku TEXT NOT NULL,
		price INTEGER NOT NULL,
		uom TEXT NOT NULL,
		alt_uom TEXT NOT NULL,
		updated_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_medicines_barcode ON medicines(barcode);
	CREATE INDEX IF NOT EXISTS idx_medicines_sku ON medicines(sku);
	CREATE INDEX IF NOT EXISTS idx_medicines_updated_at ON medicines(updated_at DESC);

	CREATE VIRTUAL TABLE IF NOT EXISTS medicines_fts USING fts5(
		name,
		sku,
		barcode,
		content='medicines',
		content_rowid='rowid',
		tokenize='unicode61'
	);

	CREATE TRIGGER IF NOT EXISTS medicines_ai AFTER INSERT ON medicines BEGIN
		INSERT INTO medicines_fts(rowid, name, sku, barcode)
		VALUES (new.rowid, new.name, new.sku, new.barcode);
	END;
	CREATE TRIGGER IF NOT EXISTS medicines_ad AFTER DELETE ON medicines BEGIN
		INSERT INTO medicines_fts(medicines_fts, rowid, name, sku, barcode)
		VALUES ('delete', old.rowid, old.name, old.sku, old.barcode);
	END;
	CREATE TRIGGER IF NOT EXISTS medicines_au AFTER UPDATE ON medicines BEGIN
		INSERT INTO medicines_fts(medicines_fts, rowid, name, sku, barcode)
		VALUES ('delete', old.rowid, old.name, old.sku, old.barcode);
		INSERT INTO medicines_fts(rowid, name, sku, barcode)
		VALUES (new.rowid, new.name, new.sku, new.barcode);
	END;`

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) ReplaceAll(ctx context.Context, meds []domain.Medicine) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM medicines`); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO medicines (id, name, name_lower, barcode, sku, price, uom, alt_uom, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range meds {
		if _, err := stmt.ExecContext(
			ctx,
			m.ID,
			m.Name,
			strings.ToLower(m.Name),
			m.Barcode,
			m.SKU,
			m.Price,
			m.UOM,
			m.AltUOM,
			m.UpdatedAt.UnixNano(),
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) FindByBarcode(ctx context.Context, barcode string) (domain.Medicine, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, barcode, sku, price, uom, alt_uom, updated_at
		FROM medicines
		WHERE barcode = ?
		LIMIT 1
	`, barcode)
	return scanMedicineRow(row)
}

func (s *SQLiteStore) SearchFTS(ctx context.Context, query string, limit, offset int) (port.MedicinePage, error) {
	if limit <= 0 {
		limit = 20
	}
	ftsQuery := buildFTSPrefixQuery(query)
	if ftsQuery == "" {
		return port.MedicinePage{}, nil
	}

	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM medicines_fts
		WHERE medicines_fts MATCH ?
	`, ftsQuery).Scan(&total)
	if err != nil {
		return port.MedicinePage{}, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.name, m.barcode, m.sku, m.price, m.uom, m.alt_uom, m.updated_at
		FROM medicines_fts f
		JOIN medicines m ON m.rowid = f.rowid
		WHERE medicines_fts MATCH ?
		ORDER BY m.name ASC
		LIMIT ? OFFSET ?
	`, ftsQuery, limit, offset)
	if err != nil {
		return port.MedicinePage{}, err
	}
	defer rows.Close()

	items, err := scanMedicineRows(rows)
	if err != nil {
		return port.MedicinePage{}, err
	}
	return port.MedicinePage{Items: items, Total: total}, nil
}

func (s *SQLiteStore) SearchLike(ctx context.Context, query string, limit, offset int) (port.MedicinePage, error) {
	if limit <= 0 {
		limit = 20
	}
	tokens := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(tokens) == 0 {
		return port.MedicinePage{}, nil
	}

	clauses := make([]string, 0, len(tokens))
	args := make([]any, 0, len(tokens)+2)
	for _, tok := range tokens {
		clauses = append(clauses, "name_lower LIKE ?")
		args = append(args, "%"+tok+"%")
	}
	where := strings.Join(clauses, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM medicines WHERE "+where, args...).Scan(&total); err != nil {
		return port.MedicinePage{}, err
	}

	argsWithPage := append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, barcode, sku, price, uom, alt_uom, updated_at
		FROM medicines
		WHERE `+where+`
		ORDER BY name ASC
		LIMIT ? OFFSET ?
	`, argsWithPage...)
	if err != nil {
		return port.MedicinePage{}, err
	}
	defer rows.Close()

	items, err := scanMedicineRows(rows)
	if err != nil {
		return port.MedicinePage{}, err
	}
	return port.MedicinePage{Items: items, Total: total}, nil
}

func (s *SQLiteStore) ListRecent(ctx context.Context, since time.Time, limit, offset int) (port.MedicinePage, error) {
	if limit <= 0 {
		limit = 20
	}
	sinceNano := since.UnixNano()

	var total int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM medicines WHERE updated_at >= ?
	`, sinceNano).Scan(&total); err != nil {
		return port.MedicinePage{}, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, barcode, sku, price, uom, alt_uom, updated_at
		FROM medicines
		WHERE updated_at >= ?
		ORDER BY updated_at DESC, name ASC
		LIMIT ? OFFSET ?
	`, sinceNano, limit, offset)
	if err != nil {
		return port.MedicinePage{}, err
	}
	defer rows.Close()

	items, err := scanMedicineRows(rows)
	if err != nil {
		return port.MedicinePage{}, err
	}
	return port.MedicinePage{Items: items, Total: total}, nil
}

func buildFTSPrefixQuery(input string) string {
	tokens := strings.Fields(strings.TrimSpace(input))
	if len(tokens) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		clean := strings.ReplaceAll(tok, `"`, "")
		if clean == "" {
			continue
		}
		parts = append(parts, `"`+clean+`"*`)
	}
	return strings.Join(parts, " ")
}

func scanMedicineRow(row *sql.Row) (domain.Medicine, error) {
	var m domain.Medicine
	var updatedNano int64
	if err := row.Scan(&m.ID, &m.Name, &m.Barcode, &m.SKU, &m.Price, &m.UOM, &m.AltUOM, &updatedNano); err != nil {
		if err == sql.ErrNoRows {
			return domain.Medicine{}, domain.ErrNotFound
		}
		return domain.Medicine{}, err
	}
	m.UpdatedAt = time.Unix(0, updatedNano).UTC()
	return m, nil
}

func scanMedicineRows(rows *sql.Rows) ([]domain.Medicine, error) {
	var out []domain.Medicine
	for rows.Next() {
		var m domain.Medicine
		var updatedNano int64
		if err := rows.Scan(&m.ID, &m.Name, &m.Barcode, &m.SKU, &m.Price, &m.UOM, &m.AltUOM, &updatedNano); err != nil {
			return nil, err
		}
		m.UpdatedAt = time.Unix(0, updatedNano).UTC()
		out = append(out, m)
	}
	return out, rows.Err()
}
