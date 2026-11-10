package spike

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
)

func generateSampleMedicines(count int, now time.Time) []domain.Medicine {
	bases := []string{
		"Paracetamol 500mg Tablet",
		"Amoxicillin 500mg Kapsul",
		"Omeprazole 20mg Kapsul",
		"Cetirizine 10mg Tablet",
		"Antasida Doen Suspensi 60ml",
		"OBH Combi Batuk Plus Flu 100ml",
		"Vitamin C 500mg Tablet",
		"Ibuprofen 400mg Tablet",
		"Metformin 500mg Tablet",
		"Amlodipine 5mg Tablet",
	}
	uoms := []string{"Strip", "Box", "Botol", "Tube", "Sachet"}
	altUoms := []string{"Tablet", "Kapsul", "ml", "gram", "Pcs"}

	meds := make([]domain.Medicine, 0, count)
	for i := 0; i < count; i++ {
		base := bases[i%len(bases)]
		daysAgo := (i % 90)
		// 5% data diupdate dalam 7 hari terakhir
		if i%20 == 0 {
			daysAgo = i % 7
		}
		meds = append(meds, domain.Medicine{
			ID:        fmt.Sprintf("MED-%05d", i),
			Name:      fmt.Sprintf("%s Varian %04d", base, i),
			Barcode:   fmt.Sprintf("8991234%06d", i),
			SKU:       fmt.Sprintf("GF-%05d", i),
			Price:     int64(2500 + (i%50)*500),
			UOM:       uoms[i%len(uoms)],
			AltUOM:    altUoms[i%len(altUoms)],
			UpdatedAt: now.Add(-time.Duration(daysAgo) * 24 * time.Hour),
		})
	}
	return meds
}

func TestSpikeStoresParity(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	meds := generateSampleMedicines(500, now)

	tmp := t.TempDir()
	sqliteStore, err := NewSQLiteStore(filepath.Join(tmp, "test.db"))
	if err != nil {
		t.Fatalf("new sqlite: %v", err)
	}
	defer sqliteStore.Close()

	bboltStore, err := NewBBoltStore(filepath.Join(tmp, "test.bbolt"))
	if err != nil {
		t.Fatalf("new bbolt: %v", err)
	}
	defer bboltStore.Close()

	if err := sqliteStore.ReplaceAll(ctx, meds); err != nil {
		t.Fatalf("sqlite replace: %v", err)
	}
	if err := bboltStore.ReplaceAll(ctx, meds); err != nil {
		t.Fatalf("bbolt replace: %v", err)
	}

	// 1. Barcode lookup
	m1, err := sqliteStore.FindByBarcode(ctx, "8991234000123")
	if err != nil {
		t.Fatalf("sqlite barcode: %v", err)
	}
	m2, err := bboltStore.FindByBarcode(ctx, "8991234000123")
	if err != nil {
		t.Fatalf("bbolt barcode: %v", err)
	}
	if m1.ID != m2.ID || m1.Price != m2.Price {
		t.Fatalf("barcode mismatch: sqlite=%+v bbolt=%+v", m1, m2)
	}

	// 2. Prefix search
	pFTS, err := sqliteStore.SearchFTS(ctx, "parace 500", 20, 0)
	if err != nil {
		t.Fatalf("sqlite fts: %v", err)
	}
	pLike, err := sqliteStore.SearchLike(ctx, "parace 500", 20, 0)
	if err != nil {
		t.Fatalf("sqlite like: %v", err)
	}
	pBolt, err := bboltStore.SearchByName(ctx, "parace 500", 20, 0)
	if err != nil {
		t.Fatalf("bbolt search: %v", err)
	}
	if pFTS.Total == 0 || pFTS.Total != pLike.Total || pFTS.Total != pBolt.Total {
		t.Fatalf("search total mismatch: fts=%d like=%d bbolt=%d", pFTS.Total, pLike.Total, pBolt.Total)
	}

	// 3. Mid-word search (OCR fallback case)
	pMidLike, err := sqliteStore.SearchLike(ctx, "cetamol", 20, 0)
	if err != nil {
		t.Fatalf("sqlite mid like: %v", err)
	}
	pMidBolt, err := bboltStore.SearchByName(ctx, "cetamol", 20, 0)
	if err != nil {
		t.Fatalf("bbolt mid: %v", err)
	}
	if pMidLike.Total == 0 || pMidLike.Total != pMidBolt.Total {
		t.Fatalf("mid-word mismatch: like=%d bbolt=%d", pMidLike.Total, pMidBolt.Total)
	}

	// 4. Recent updates
	since := now.Add(-7 * 24 * time.Hour)
	rSQL, err := sqliteStore.ListRecent(ctx, since, 20, 0)
	if err != nil {
		t.Fatalf("sqlite recent: %v", err)
	}
	rBolt, err := bboltStore.ListRecent(ctx, since, 20, 0)
	if err != nil {
		t.Fatalf("bbolt recent: %v", err)
	}
	if rSQL.Total == 0 || rSQL.Total != rBolt.Total {
		t.Fatalf("recent mismatch: sqlite=%d bbolt=%d", rSQL.Total, rBolt.Total)
	}
}

func setupBenchStores(b *testing.B) (*SQLiteStore, *BBoltStore, []domain.Medicine, time.Time) {
	b.Helper()
	ctx := context.Background()
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	meds := generateSampleMedicines(5000, now)
	tmp := b.TempDir()

	sqliteStore, err := NewSQLiteStore(filepath.Join(tmp, "bench.db"))
	if err != nil {
		b.Fatalf("new sqlite: %v", err)
	}
	b.Cleanup(func() { _ = sqliteStore.Close() })

	bboltStore, err := NewBBoltStore(filepath.Join(tmp, "bench.bbolt"))
	if err != nil {
		b.Fatalf("new bbolt: %v", err)
	}
	b.Cleanup(func() { _ = bboltStore.Close() })

	if err := sqliteStore.ReplaceAll(ctx, meds); err != nil {
		b.Fatalf("seed sqlite: %v", err)
	}
	if err := bboltStore.ReplaceAll(ctx, meds); err != nil {
		b.Fatalf("seed bbolt: %v", err)
	}

	return sqliteStore, bboltStore, meds, now
}

func BenchmarkSQLite_ReplaceAll5000(b *testing.B) {
	sqliteStore, _, meds, _ := setupBenchStores(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := sqliteStore.ReplaceAll(ctx, meds); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBBolt_ReplaceAll5000(b *testing.B) {
	_, bboltStore, meds, _ := setupBenchStores(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := bboltStore.ReplaceAll(ctx, meds); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLite_FindByBarcode(b *testing.B) {
	sqliteStore, _, _, _ := setupBenchStores(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := sqliteStore.FindByBarcode(ctx, "8991234002500"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBBolt_FindByBarcode(b *testing.B) {
	_, bboltStore, _, _ := setupBenchStores(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := bboltStore.FindByBarcode(ctx, "8991234002500"); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLite_SearchNameFTS(b *testing.B) {
	sqliteStore, _, _, _ := setupBenchStores(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := sqliteStore.SearchFTS(ctx, "parace 500", 20, 0); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLite_SearchNameLike(b *testing.B) {
	sqliteStore, _, _, _ := setupBenchStores(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := sqliteStore.SearchLike(ctx, "cetamol", 20, 0); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBBolt_SearchNameScan(b *testing.B) {
	_, bboltStore, _, _ := setupBenchStores(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := bboltStore.SearchByName(ctx, "parace 500", 20, 0); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLite_ListRecent20(b *testing.B) {
	sqliteStore, _, _, now := setupBenchStores(b)
	ctx := context.Background()
	since := now.Add(-7 * 24 * time.Hour)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := sqliteStore.ListRecent(ctx, since, 20, 0); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBBolt_ListRecent20(b *testing.B) {
	_, bboltStore, _, now := setupBenchStores(b)
	ctx := context.Background()
	since := now.Add(-7 * 24 * time.Hour)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := bboltStore.ListRecent(ctx, since, 20, 0); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSQLite_ParallelSearchFTS(b *testing.B) {
	sqliteStore, _, _, _ := setupBenchStores(b)
	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := sqliteStore.SearchFTS(ctx, "amox 500", 20, 0); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkBBolt_ParallelSearchScan(b *testing.B) {
	_, bboltStore, _, _ := setupBenchStores(b)
	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := bboltStore.SearchByName(ctx, "amox 500", 20, 0); err != nil {
				b.Fatal(err)
			}
		}
	})
}
