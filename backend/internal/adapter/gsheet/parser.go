package gsheet

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
)

var defaultMedicineHeaders = []string{"id", "name", "barcode", "sku", "price", "uom", "alt_uom", "updated_at"}

// ParseMedicinesSheet mem-parsing baris Google Sheet menjadi slice Medicine berbasis header baris pertama.
func ParseMedicinesSheet(rows [][]any, fallbackTime time.Time) ([]domain.Medicine, error) {
	if len(rows) <= 1 {
		return []domain.Medicine{}, nil
	}
	colIdx := indexColumns(rows[0], medicineHeaderAliases)
	if _, ok := colIdx["name"]; !ok {
		return nil, fmt.Errorf("%w: kolom 'name' tidak ditemukan pada header sheet obat", domain.ErrInvalidInput)
	}

	out := make([]domain.Medicine, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		name := cellString(row, colIdx, "name")
		id := cellString(row, colIdx, "id")
		if name == "" && id == "" {
			continue
		}
		if name == "" {
			continue
		}

		sku := cellString(row, colIdx, "sku")
		barcode := cellString(row, colIdx, "barcode")
		if id == "" {
			switch {
			case sku != "":
				id = "sku:" + sku
			case barcode != "":
				id = "bc:" + barcode
			default:
				id = fmt.Sprintf("row:%d", i+1)
			}
		}

		price, err := ParsePrice(cellRaw(row, colIdx, "price"))
		if err != nil {
			return nil, fmt.Errorf("baris %d (%s): %w", i+1, name, err)
		}

		uom := cellString(row, colIdx, "uom")
		if uom == "" {
			uom = "Pcs"
		}
		altUOM := cellString(row, colIdx, "alt_uom")

		updatedAt, hasTime := ParseTimestamp(cellRaw(row, colIdx, "updated_at"))
		if !hasTime {
			updatedAt = fallbackTime
		}

		out = append(out, domain.Medicine{
			ID:        id,
			Name:      name,
			Barcode:   barcode,
			SKU:       sku,
			Price:     price,
			UOM:       uom,
			AltUOM:    altUOM,
			UpdatedAt: updatedAt,
		})
	}
	return out, nil
}

// MedicineToRow memetakan Medicine ke urutan kolom header Google Sheet yang aktif
// dengan sanitasi formula injection (OWASP A03:2021).
func MedicineToRow(headers []string, med domain.Medicine) []any {
	if len(headers) == 0 {
		headers = defaultMedicineHeaders
	}
	row := make([]any, len(headers))
	for i, h := range headers {
		canonical := normalizeHeader(h, medicineHeaderAliases)
		switch canonical {
		case "id":
			row[i] = sanitizeSheetCell(med.ID)
		case "name":
			row[i] = sanitizeSheetCell(med.Name)
		case "barcode":
			row[i] = sanitizeSheetCell(med.Barcode)
		case "sku":
			row[i] = sanitizeSheetCell(med.SKU)
		case "price":
			row[i] = med.Price
		case "uom":
			row[i] = sanitizeSheetCell(med.UOM)
		case "alt_uom":
			row[i] = sanitizeSheetCell(med.AltUOM)
		case "updated_at":
			row[i] = med.UpdatedAt.UTC().Format(time.RFC3339)
		default:
			row[i] = ""
		}
	}
	return row
}

// sanitizeSheetCell menetralkan karakter awal formula spreadsheet (=, +, -, @, tab, CR)
// agar tidak dieksekusi sebagai formula di Google Sheets / Excel.
func sanitizeSheetCell(s string) string {
	if s == "" {
		return ""
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	default:
		return s
	}
}

// ParseAdminsSheet mem-parsing sheet admins menjadi daftar domain.Admin.
func ParseAdminsSheet(rows [][]any) ([]domain.Admin, error) {
	if len(rows) <= 1 {
		return []domain.Admin{}, nil
	}
	colIdx := indexColumns(rows[0], adminHeaderAliases)
	if _, ok := colIdx["email"]; !ok {
		return nil, fmt.Errorf("%w: kolom 'email' tidak ditemukan pada header sheet admins", domain.ErrInvalidInput)
	}

	out := make([]domain.Admin, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		email := strings.ToLower(cellString(row, colIdx, "email"))
		if email == "" {
			continue
		}
		role := cellString(row, colIdx, "role")
		if role == "" {
			role = "admin"
		}
		out = append(out, domain.Admin{
			Email:  email,
			Name:   cellString(row, colIdx, "name"),
			Role:   role,
			Active: ParseBool(cellRaw(row, colIdx, "active")),
		})
	}
	return out, nil
}

// ParsePrice mengonversi nilai sel harga (angka atau format Rupiah seperti "Rp 15.000") menjadi int64.
func ParsePrice(raw any) (int64, error) {
	if raw == nil {
		return 0, nil
	}
	switch v := raw.(type) {
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case float64:
		return int64(math.Round(v)), nil
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, nil
		}
		s = strings.TrimPrefix(strings.ToLower(s), "rp")
		s = strings.TrimSpace(s)

		// Hilangkan akhiran desimal .00 atau ,00 bila ada
		if idx := strings.LastIndexAny(s, ".,"); idx != -1 && len(s)-idx == 3 {
			suffix := s[idx+1:]
			if suffix == "00" {
				s = s[:idx]
			}
		}

		var b strings.Builder
		for _, r := range s {
			if (r >= '0' && r <= '9') || r == '-' {
				b.WriteRune(r)
			}
		}
		cleaned := b.String()
		if cleaned == "" || cleaned == "-" {
			return 0, fmt.Errorf("%w: harga tidak valid %q", domain.ErrInvalidInput, v)
		}
		n, err := strconv.ParseInt(cleaned, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: gagal parse harga %q: %v", domain.ErrInvalidInput, v, err)
		}
		return n, nil
	default:
		return ParsePrice(fmt.Sprint(v))
	}
}

// ParseTimestamp mem-parsing waktu dari berbagai format umum Google Sheet.
func ParseTimestamp(raw any) (time.Time, bool) {
	if raw == nil {
		return time.Time{}, false
	}
	s := strings.TrimSpace(fmt.Sprint(raw))
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

// ParseBool menafsirkan status aktif dari sheet (TRUE, 1, yes, ya, aktif).
func ParseBool(raw any) bool {
	if raw == nil {
		return false
	}
	if b, ok := raw.(bool); ok {
		return b
	}
	switch strings.ToLower(strings.TrimSpace(fmt.Sprint(raw))) {
	case "true", "1", "yes", "ya", "aktif", "active", "y":
		return true
	default:
		return false
	}
}

var medicineHeaderAliases = map[string]string{
	"id":                "id",
	"name":              "name",
	"nama":              "name",
	"nama_obat":         "name",
	"barcode":           "barcode",
	"sku":               "sku",
	"kode":              "sku",
	"price":             "price",
	"harga":             "price",
	"uom":               "uom",
	"satuan":            "uom",
	"alt_uom":           "alt_uom",
	"satuan_alternatif": "alt_uom",
	"updated_at":        "updated_at",
}

var adminHeaderAliases = map[string]string{
	"email":  "email",
	"name":   "name",
	"nama":   "name",
	"role":   "role",
	"active": "active",
	"aktif":  "active",
}

func normalizeHeader(raw string, aliases map[string]string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	key = strings.ReplaceAll(key, " ", "_")
	if canonical, ok := aliases[key]; ok {
		return canonical
	}
	return key
}

func indexColumns(headerRow []any, aliases map[string]string) map[string]int {
	idx := make(map[string]int, len(headerRow))
	for i, cell := range headerRow {
		canonical := normalizeHeader(fmt.Sprint(cell), aliases)
		if canonical != "" {
			idx[canonical] = i
		}
	}
	return idx
}

func cellRaw(row []any, colIdx map[string]int, col string) any {
	i, ok := colIdx[col]
	if !ok || i < 0 || i >= len(row) {
		return nil
	}
	return row[i]
}

func cellString(row []any, colIdx map[string]int, col string) string {
	raw := cellRaw(row, colIdx, col)
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}
