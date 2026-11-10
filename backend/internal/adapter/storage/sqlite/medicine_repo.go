package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
)

type MedicineRepository struct {
	db *sql.DB
}

func NewMedicineRepository(db *sql.DB) *MedicineRepository {
	return &MedicineRepository{db: db}
}

func (r *MedicineRepository) Search(ctx context.Context, q port.MedicineSearchQuery) (port.MedicinePage, error) {
	limit := normalizeLimit(q.Limit)
	offset := normalizeOffset(q.Offset)

	if barcode := strings.TrimSpace(q.Barcode); barcode != "" {
		return r.queryPage(ctx, "WHERE barcode = ?", []any{barcode}, "name ASC", limit, offset)
	}
	if sku := strings.TrimSpace(q.SKU); sku != "" {
		return r.queryPage(ctx, "WHERE LOWER(sku) = LOWER(?)", []any{sku}, "name ASC", limit, offset)
	}

	rawQuery := strings.TrimSpace(q.Query)
	if rawQuery == "" {
		return r.queryPage(ctx, "", nil, "name ASC", limit, offset)
	}

	if ftsQuery := buildFTSPrefixQuery(rawQuery); ftsQuery != "" {
		page, err := r.searchFTS(ctx, ftsQuery, limit, offset)
		if err == nil && page.Total > 0 {
			return page, nil
		}
	}

	return r.searchLike(ctx, rawQuery, limit, offset)
}

func (r *MedicineRepository) searchFTS(ctx context.Context, ftsQuery string, limit, offset int) (port.MedicinePage, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM medicines_fts WHERE medicines_fts MATCH ?
	`, ftsQuery).Scan(&total); err != nil {
		return port.MedicinePage{}, err
	}
	if total == 0 {
		return port.MedicinePage{Items: []domain.Medicine{}, Total: 0}, nil
	}

	rows, err := r.db.QueryContext(ctx, `
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

	items, err := scanMedicines(rows)
	if err != nil {
		return port.MedicinePage{}, err
	}
	return port.MedicinePage{Items: items, Total: total}, nil
}

func (r *MedicineRepository) searchLike(ctx context.Context, query string, limit, offset int) (port.MedicinePage, error) {
	tokens := strings.Fields(strings.ToLower(query))
	if len(tokens) == 0 {
		return port.MedicinePage{Items: []domain.Medicine{}, Total: 0}, nil
	}

	clauses := make([]string, 0, len(tokens))
	args := make([]any, 0, len(tokens)*3)
	for _, tok := range tokens {
		pat := "%" + escapeLike(tok) + "%"
		clauses = append(clauses, `(name_lower LIKE ? ESCAPE '\' OR LOWER(sku) LIKE ? ESCAPE '\' OR barcode LIKE ? ESCAPE '\')`)
		args = append(args, pat, pat, pat)
	}
	where := "WHERE " + strings.Join(clauses, " AND ")
	return r.queryPage(ctx, where, args, "name ASC", limit, offset)
}

func (r *MedicineRepository) queryPage(ctx context.Context, where string, args []any, orderBy string, limit, offset int) (port.MedicinePage, error) {
	var total int
	countSQL := "SELECT COUNT(*) FROM medicines " + where
	if err := r.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return port.MedicinePage{}, fmt.Errorf("count medicines: %w", err)
	}
	if total == 0 {
		return port.MedicinePage{Items: []domain.Medicine{}, Total: 0}, nil
	}

	queryArgs := append(append([]any{}, args...), limit, offset)
	selectSQL := `
		SELECT id, name, barcode, sku, price, uom, alt_uom, updated_at
		FROM medicines ` + where + `
		ORDER BY ` + orderBy + `
		LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, selectSQL, queryArgs...)
	if err != nil {
		return port.MedicinePage{}, fmt.Errorf("select medicines: %w", err)
	}
	defer rows.Close()

	items, err := scanMedicines(rows)
	if err != nil {
		return port.MedicinePage{}, err
	}
	return port.MedicinePage{Items: items, Total: total}, nil
}

func (r *MedicineRepository) FindByID(ctx context.Context, id string) (domain.Medicine, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, barcode, sku, price, uom, alt_uom, updated_at
		FROM medicines WHERE id = ?
	`, id)
	return scanMedicine(row)
}

func (r *MedicineRepository) FindByBarcode(ctx context.Context, barcode string) (domain.Medicine, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, barcode, sku, price, uom, alt_uom, updated_at
		FROM medicines WHERE barcode = ? LIMIT 1
	`, strings.TrimSpace(barcode))
	return scanMedicine(row)
}

func (r *MedicineRepository) ListAll(ctx context.Context) ([]domain.Medicine, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, barcode, sku, price, uom, alt_uom, updated_at
		FROM medicines ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list all medicines: %w", err)
	}
	defer rows.Close()
	return scanMedicines(rows)
}

func (r *MedicineRepository) ListRecent(ctx context.Context, since time.Time, limit, offset int) (port.MedicinePage, error) {
	limit = normalizeLimit(limit)
	offset = normalizeOffset(offset)
	return r.queryPage(ctx, "WHERE updated_at >= ?", []any{since.UnixNano()}, "updated_at DESC, name ASC", limit, offset)
}

func (r *MedicineRepository) Upsert(ctx context.Context, med domain.Medicine) error {
	_, err := r.db.ExecContext(ctx, upsertMedicineSQL,
		med.ID,
		med.Name,
		strings.ToLower(med.Name),
		med.Barcode,
		med.SKU,
		med.Price,
		med.UOM,
		med.AltUOM,
		med.UpdatedAt.UnixNano(),
	)
	if err != nil {
		return fmt.Errorf("upsert medicine %s: %w", med.ID, err)
	}
	return nil
}

func (r *MedicineRepository) ApplySyncDiff(ctx context.Context, toUpsert []domain.Medicine, toDeleteIDs []string) error {
	if len(toUpsert) == 0 && len(toDeleteIDs) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin sync diff tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if len(toUpsert) > 0 {
		stmt, err := tx.PrepareContext(ctx, upsertMedicineSQL)
		if err != nil {
			return fmt.Errorf("prepare sync upsert: %w", err)
		}
		defer stmt.Close()

		for _, m := range toUpsert {
			if _, err := stmt.ExecContext(ctx,
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
				return fmt.Errorf("exec sync upsert %s: %w", m.ID, err)
			}
		}
	}

	if len(toDeleteIDs) > 0 {
		delStmt, err := tx.PrepareContext(ctx, `DELETE FROM medicines WHERE id = ?`)
		if err != nil {
			return fmt.Errorf("prepare sync delete: %w", err)
		}
		defer delStmt.Close()

		for _, id := range toDeleteIDs {
			if _, err := delStmt.ExecContext(ctx, id); err != nil {
				return fmt.Errorf("exec sync delete %s: %w", id, err)
			}
		}
	}

	return tx.Commit()
}

func (r *MedicineRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM medicines WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete medicine %s: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound
	}
	return nil
}

const upsertMedicineSQL = `
	INSERT INTO medicines (id, name, name_lower, barcode, sku, price, uom, alt_uom, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		name = excluded.name,
		name_lower = excluded.name_lower,
		barcode = excluded.barcode,
		sku = excluded.sku,
		price = excluded.price,
		uom = excluded.uom,
		alt_uom = excluded.alt_uom,
		updated_at = excluded.updated_at
`

func buildFTSPrefixQuery(input string) string {
	tokens := strings.Fields(strings.TrimSpace(input))
	if len(tokens) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		clean := strings.Map(func(r rune) rune {
			if r == '"' || r == '*' {
				return -1
			}
			return r
		}, tok)
		clean = strings.TrimSpace(clean)
		if clean == "" {
			continue
		}
		parts = append(parts, `"`+clean+`"*`)
	}
	return strings.Join(parts, " ")
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func scanMedicine(row *sql.Row) (domain.Medicine, error) {
	var m domain.Medicine
	var updatedNano int64
	if err := row.Scan(&m.ID, &m.Name, &m.Barcode, &m.SKU, &m.Price, &m.UOM, &m.AltUOM, &updatedNano); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Medicine{}, domain.ErrNotFound
		}
		return domain.Medicine{}, err
	}
	m.UpdatedAt = time.Unix(0, updatedNano).UTC()
	return m, nil
}

func scanMedicines(rows *sql.Rows) ([]domain.Medicine, error) {
	out := []domain.Medicine{}
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
