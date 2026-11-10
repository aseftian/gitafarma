package domain

import (
	"fmt"
	"strings"
	"time"
)

// Medicine merepresentasikan data obat beserta referensi harga pada satu unit dan unit alternatif.
type Medicine struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Barcode   string    `json:"barcode"`
	SKU       string    `json:"sku"`
	Price     int64     `json:"price"`
	UOM       string    `json:"uom"`
	AltUOM    string    `json:"alt_uom"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Validate memeriksa kelayakan field wajib obat sebelum disimpan atau disinkronkan.
func (m Medicine) Validate() error {
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("%w: nama obat wajib diisi", ErrInvalidInput)
	}
	if m.Price < 0 {
		return fmt.Errorf("%w: harga tidak boleh negatif", ErrInvalidInput)
	}
	if strings.TrimSpace(m.UOM) == "" {
		return fmt.Errorf("%w: uom wajib diisi", ErrInvalidInput)
	}
	return nil
}

// ChangeType mendefinisikan jenis perubahan harga saat sinkronisasi.
type ChangeType string

const (
	ChangeAdded        ChangeType = "added"
	ChangePriceUpdated ChangeType = "price_updated"
)

// PriceChange merepresentasikan obat yang baru ditambahkan atau mengalami perubahan harga.
type PriceChange struct {
	Medicine   Medicine   `json:"medicine"`
	OldPrice   int64      `json:"old_price"`
	NewPrice   int64      `json:"new_price"`
	ChangeType ChangeType `json:"change_type"`
}
