package gsheet

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ardisan/gitafarma/backend/internal/domain"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Client membungkus Google Sheets API untuk membaca & menulis sheet obat dan whitelist admin.
type Client struct {
	svc            *sheets.Service
	spreadsheetID  string
	medicinesSheet string
	adminsSheet    string
	writeMu        sync.Mutex
}

// NewClient membuat client Google Sheets API menggunakan file JSON Google Service Account.
func NewClient(ctx context.Context, credentialsFile, spreadsheetID, medicinesSheet, adminsSheet string) (*Client, error) {
	if strings.TrimSpace(spreadsheetID) == "" {
		return nil, domain.ErrSheetNotConfigured
	}
	svc, err := sheets.NewService(ctx, option.WithAuthCredentialsFile(option.ServiceAccount, credentialsFile), option.WithScopes(sheets.SpreadsheetsScope))
	if err != nil {
		return nil, fmt.Errorf("create google sheets service: %w", err)
	}
	return &Client{
		svc:            svc,
		spreadsheetID:  spreadsheetID,
		medicinesSheet: medicinesSheet,
		adminsSheet:    adminsSheet,
	}, nil
}

func (c *Client) FetchAllMedicines(ctx context.Context) ([]domain.Medicine, error) {
	rows, err := c.readSheetRows(ctx, c.medicinesSheet)
	if err != nil {
		return nil, err
	}
	return ParseMedicinesSheet(rows, time.Now().UTC())
}

func (c *Client) UpsertMedicine(ctx context.Context, med domain.Medicine) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	rows, err := c.readSheetRows(ctx, c.medicinesSheet)
	if err != nil {
		return err
	}

	var headers []string
	if len(rows) > 0 {
		for _, cell := range rows[0] {
			headers = append(headers, fmt.Sprint(cell))
		}
	} else {
		headers = defaultMedicineHeaders
		headerAny := make([]any, len(headers))
		for i, h := range headers {
			headerAny[i] = h
		}
		if _, err := c.svc.Spreadsheets.Values.Update(
			c.spreadsheetID,
			fmt.Sprintf("%s!A1", c.medicinesSheet),
			&sheets.ValueRange{Values: [][]any{headerAny}},
		).ValueInputOption("USER_ENTERED").Context(ctx).Do(); err != nil {
			return fmt.Errorf("write header row: %w", err)
		}
	}

	rowValues := MedicineToRow(headers, med)
	colIdx := indexColumns(stringSliceToAny(headers), medicineHeaderAliases)
	idCol, hasID := colIdx["id"]

	if hasID {
		for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
			if idCol < len(rows[rowIdx]) && strings.TrimSpace(fmt.Sprint(rows[rowIdx][idCol])) == med.ID {
				targetRange := fmt.Sprintf("%s!A%d", c.medicinesSheet, rowIdx+1)
				_, err := retryWithBackoff(ctx, defaultRetryConfig, func() (struct{}, error) {
					_, callErr := c.svc.Spreadsheets.Values.Update(
						c.spreadsheetID,
						targetRange,
						&sheets.ValueRange{Values: [][]any{rowValues}},
					).ValueInputOption("USER_ENTERED").Context(ctx).Do()
					return struct{}{}, callErr
				})
				if err != nil {
					return fmt.Errorf("update gsheet row %d: %w", rowIdx+1, err)
				}
				return nil
			}
		}
	}

	_, err = retryWithBackoff(ctx, defaultRetryConfig, func() (struct{}, error) {
		_, callErr := c.svc.Spreadsheets.Values.Append(
			c.spreadsheetID,
			fmt.Sprintf("%s!A:A", c.medicinesSheet),
			&sheets.ValueRange{Values: [][]any{rowValues}},
		).ValueInputOption("USER_ENTERED").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
		return struct{}{}, callErr
	})
	if err != nil {
		return fmt.Errorf("append gsheet medicine row: %w", err)
	}
	return nil
}

func (c *Client) DeleteMedicine(ctx context.Context, id string) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	rows, err := c.readSheetRows(ctx, c.medicinesSheet)
	if err != nil {
		return err
	}
	if len(rows) <= 1 {
		return domain.ErrNotFound
	}
	colIdx := indexColumns(rows[0], medicineHeaderAliases)
	idCol, hasID := colIdx["id"]
	if !hasID {
		return fmt.Errorf("%w: kolom 'id' tidak ditemukan untuk hapus baris", domain.ErrInvalidInput)
	}

	targetRow := -1
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		if idCol < len(rows[rowIdx]) && strings.TrimSpace(fmt.Sprint(rows[rowIdx][idCol])) == id {
			targetRow = rowIdx
			break
		}
	}
	if targetRow < 0 {
		return domain.ErrNotFound
	}

	sheetID, err := c.lookupSheetID(ctx, c.medicinesSheet)
	if err != nil {
		return err
	}

	_, err = retryWithBackoff(ctx, defaultRetryConfig, func() (struct{}, error) {
		_, callErr := c.svc.Spreadsheets.BatchUpdate(c.spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{
				{
					DeleteDimension: &sheets.DeleteDimensionRequest{
						Range: &sheets.DimensionRange{
							SheetId:    sheetID,
							Dimension:  "ROWS",
							StartIndex: int64(targetRow),
							EndIndex:   int64(targetRow + 1),
						},
					},
				},
			},
		}).Context(ctx).Do()
		return struct{}{}, callErr
	})
	if err != nil {
		return fmt.Errorf("delete gsheet row %d: %w", targetRow+1, err)
	}
	return nil
}

func (c *Client) FetchAdmins(ctx context.Context) ([]domain.Admin, error) {
	rows, err := c.readSheetRows(ctx, c.adminsSheet)
	if err != nil {
		return nil, err
	}
	return ParseAdminsSheet(rows)
}

func (c *Client) readSheetRows(ctx context.Context, sheetName string) ([][]any, error) {
	resp, err := retryWithBackoff(ctx, defaultRetryConfig, func() (*sheets.ValueRange, error) {
		return c.svc.Spreadsheets.Values.Get(c.spreadsheetID, sheetName).Context(ctx).Do()
	})
	if err != nil {
		return nil, fmt.Errorf("read sheet %s: %w", sheetName, err)
	}
	return resp.Values, nil
}

func (c *Client) lookupSheetID(ctx context.Context, sheetName string) (int64, error) {
	ss, err := retryWithBackoff(ctx, defaultRetryConfig, func() (*sheets.Spreadsheet, error) {
		return c.svc.Spreadsheets.Get(c.spreadsheetID).Fields("sheets.properties").Context(ctx).Do()
	})
	if err != nil {
		return 0, fmt.Errorf("get spreadsheet metadata: %w", err)
	}
	for _, s := range ss.Sheets {
		if s.Properties != nil && strings.EqualFold(s.Properties.Title, sheetName) {
			return s.Properties.SheetId, nil
		}
	}
	return 0, fmt.Errorf("%w: sheet %s tidak ditemukan", domain.ErrNotFound, sheetName)
}

func stringSliceToAny(in []string) []any {
	out := make([]any, len(in))
	for i, s := range in {
		out[i] = s
	}
	return out
}

// UnconfiguredGateway dipakai ketika spreadsheet ID belum diisi (mis. dev lokal tanpa kredensial).
type UnconfiguredGateway struct{}

func (UnconfiguredGateway) FetchAllMedicines(context.Context) ([]domain.Medicine, error) {
	return nil, domain.ErrSheetNotConfigured
}
func (UnconfiguredGateway) UpsertMedicine(context.Context, domain.Medicine) error {
	return domain.ErrSheetNotConfigured
}
func (UnconfiguredGateway) DeleteMedicine(context.Context, string) error {
	return domain.ErrSheetNotConfigured
}
func (UnconfiguredGateway) FetchAdmins(context.Context) ([]domain.Admin, error) {
	return nil, domain.ErrSheetNotConfigured
}
