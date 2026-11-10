package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Open membuka koneksi SQLite dalam mode WAL dan memastikan skema tabel sudah terinisialisasi.
func Open(dsn string) (*sql.DB, error) {
	if dsn != ":memory:" && !strings.HasPrefix(dsn, "file:") {
		dir := filepath.Dir(dsn)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("create sqlite dir %s: %w", dir, err)
			}
		}
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(4)

	pragmas := []string{
		"PRAGMA journal_mode=WAL;",
		"PRAGMA synchronous=NORMAL;",
		"PRAGMA busy_timeout=5000;",
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

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	const ddl = `
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
	CREATE INDEX IF NOT EXISTS idx_medicines_sku_lower ON medicines(LOWER(sku));
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
	END;

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		description TEXT NOT NULL,
		updated_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS push_subscriptions (
		endpoint TEXT PRIMARY KEY,
		p256dh TEXT NOT NULL,
		auth TEXT NOT NULL,
		user_agent TEXT NOT NULL,
		created_at INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		admin_email TEXT NOT NULL,
		action TEXT NOT NULL,
		entity_type TEXT NOT NULL,
		entity_id TEXT NOT NULL,
		detail TEXT NOT NULL,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);
	`
	if _, err := db.Exec(ddl); err != nil {
		return fmt.Errorf("migrate sqlite schema: %w", err)
	}
	return nil
}
