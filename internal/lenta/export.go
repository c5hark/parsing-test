package lenta

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func ExportToJSON(products []ProductExport, path string) error {
	if err := os.MkdirAll(path, 0755); err != nil && path != "" {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(products)
}

func ExportToCSV(products []ProductExport, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil && path != "" {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Comma = ';'
	if err := w.Write([]string{"name", "price", "url"}); err != nil {
		return err
	}
	for _, p := range products {
		if err := w.Write([]string{p.Name, fmtPrice(p.Price), p.URL}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func fmtPrice(rub float64) string {
	return fmt.Sprintf("%.2f", rub)
}
