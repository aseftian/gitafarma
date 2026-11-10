package spike

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"github.com/ardisan/gitafarma/backend/internal/port"
	"go.etcd.io/bbolt"
)

var (
	bucketMedicines    = []byte("medicines")
	bucketIdxBarcode   = []byte("idx_barcode")
	bucketIdxUpdatedAt = []byte("idx_updated_at")
)

type BBoltStore struct {
	db *bbolt.DB
}

func NewBBoltStore(path string) (*BBoltStore, error) {
	db, err := bbolt.Open(path, 0o600, &bbolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open bbolt: %w", err)
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		for _, b := range [][]byte{bucketMedicines, bucketIdxBarcode, bucketIdxUpdatedAt} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("init bbolt buckets: %w", err)
	}

	return &BBoltStore{db: db}, nil
}

func (s *BBoltStore) Close() error {
	return s.db.Close()
}

func (s *BBoltStore) ReplaceAll(_ context.Context, meds []domain.Medicine) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		for _, b := range [][]byte{bucketMedicines, bucketIdxBarcode, bucketIdxUpdatedAt} {
			if err := tx.DeleteBucket(b); err != nil && err != bbolt.ErrBucketNotFound {
				return err
			}
			if _, err := tx.CreateBucket(b); err != nil {
				return err
			}
		}

		bMeds := tx.Bucket(bucketMedicines)
		bBarcode := tx.Bucket(bucketIdxBarcode)
		bUpdated := tx.Bucket(bucketIdxUpdatedAt)

		for _, m := range meds {
			raw, err := json.Marshal(m)
			if err != nil {
				return err
			}
			idBytes := []byte(m.ID)
			if err := bMeds.Put(idBytes, raw); err != nil {
				return err
			}
			if m.Barcode != "" {
				if err := bBarcode.Put([]byte(m.Barcode), idBytes); err != nil {
					return err
				}
			}
			timeKey := updatedIndexKey(m.UpdatedAt, m.ID)
			if err := bUpdated.Put(timeKey, idBytes); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *BBoltStore) FindByBarcode(_ context.Context, barcode string) (domain.Medicine, error) {
	var med domain.Medicine
	err := s.db.View(func(tx *bbolt.Tx) error {
		id := tx.Bucket(bucketIdxBarcode).Get([]byte(barcode))
		if id == nil {
			return domain.ErrNotFound
		}
		raw := tx.Bucket(bucketMedicines).Get(id)
		if raw == nil {
			return domain.ErrNotFound
		}
		return json.Unmarshal(raw, &med)
	})
	return med, err
}

func (s *BBoltStore) SearchByName(_ context.Context, query string, limit, offset int) (port.MedicinePage, error) {
	if limit <= 0 {
		limit = 20
	}
	tokens := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	if len(tokens) == 0 {
		return port.MedicinePage{}, nil
	}

	var matched []domain.Medicine
	err := s.db.View(func(tx *bbolt.Tx) error {
		c := tx.Bucket(bucketMedicines).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var m domain.Medicine
			if err := json.Unmarshal(v, &m); err != nil {
				return err
			}
			lowerName := strings.ToLower(m.Name)
			ok := true
			for _, tok := range tokens {
				if !strings.Contains(lowerName, tok) {
					ok = false
					break
				}
			}
			if ok {
				matched = append(matched, m)
			}
		}
		return nil
	})
	if err != nil {
		return port.MedicinePage{}, err
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Name < matched[j].Name
	})

	total := len(matched)
	if offset >= total {
		return port.MedicinePage{Total: total}, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return port.MedicinePage{
		Items: matched[offset:end],
		Total: total,
	}, nil
}

func (s *BBoltStore) ListRecent(_ context.Context, since time.Time, limit, offset int) (port.MedicinePage, error) {
	if limit <= 0 {
		limit = 20
	}
	minKey := []byte(fmt.Sprintf("%020d|", since.UnixNano()))

	var all []domain.Medicine
	err := s.db.View(func(tx *bbolt.Tx) error {
		bUpdated := tx.Bucket(bucketIdxUpdatedAt)
		bMeds := tx.Bucket(bucketMedicines)

		c := bUpdated.Cursor()
		for k, id := c.Last(); k != nil; k, id = c.Prev() {
			if bytes.Compare(k, minKey) < 0 {
				break
			}
			raw := bMeds.Get(id)
			if raw == nil {
				continue
			}
			var m domain.Medicine
			if err := json.Unmarshal(raw, &m); err != nil {
				return err
			}
			all = append(all, m)
		}
		return nil
	})
	if err != nil {
		return port.MedicinePage{}, err
	}

	total := len(all)
	if offset >= total {
		return port.MedicinePage{Total: total}, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return port.MedicinePage{
		Items: all[offset:end],
		Total: total,
	}, nil
}

func updatedIndexKey(t time.Time, id string) []byte {
	return []byte(fmt.Sprintf("%020d|%s", t.UnixNano(), id))
}
