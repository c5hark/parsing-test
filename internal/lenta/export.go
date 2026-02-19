package lenta

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
)

// ExportToCSV сохраняет товары в CSV.
// Используется разделитель ';' (совместимость с RU Excel).
// Файл перезаписывается, если существует.

func ExportToCSV(products []ProductExport, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil && path != "" {
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
