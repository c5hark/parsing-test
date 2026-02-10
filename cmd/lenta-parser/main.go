package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"testJob/internal/lenta"
	"time"
)

func parseCategoryIDs(s string) ([]int, error) {
	if s == "" {
		return nil, fmt.Errorf("пустой список категорий")
	}
	parts := strings.Split(s, ",")
	var ids []int
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("не удалось распарсить category id %q: %w", p, err)
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("не распарсили ни одной id категории")
	}
	return ids, nil
}

func main() {
	proxy := flag.String("proxy", "", "прокси, например http://user:pass@host:port")
	sessionToken := flag.String("session-token", "", "токкен сессии Lenta (Utk_SessionToken)")
	storeID := flag.Int("store-id", 104, "id магазина")
	domain := flag.String("domain", "moscow", "регион (x-domain) например msocow")
	deviceID := flag.String("device-id", "", "x-device-id (если пусто - генерируется один раз)")
	userSessionID := flag.String("user-session-id", "", "UserSessionId из cookie")
	outputPath := flag.String("out", "lenta_products.json", "файл выгрузки (JSON или .csv)")
	categoryIDstr := flag.String("categories", "2.5", "id категорий через запятую (например 2=Сыры)")

	flag.Parse()

	if *sessionToken == "" {
		log.Fatal("укажите -session-token")
	}
	if *proxy == "" {
		log.Fatal("укажите -proxy")
	}

	categoryIDs, err := parseCategoryIDs(*categoryIDstr)
	if err != nil {
		log.Fatal("ошибка парсинга категорий: %v", err)
	}

	cfg := &lenta.Config{
		ProxyURL:      *proxy,
		SessionToken:  *sessionToken,
		StoreID:       *storeID,
		Domain:        *domain,
		DeviceID:      *deviceID,
		UserSessionID: *userSessionID,
		OutputPath:    *outputPath,
		CategoryIDs:   categoryIDs,
	}

	client, err := lenta.NewClient(cfg)
	if err != nil {
		log.Fatal("создание клиента: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var all []lenta.ProductExport
	const pagesize = 40

	for _, catID := range cfg.CategoryIDs {
		offset := 0
		for {
			resp, err := lenta.GetCategoryItems(ctx, client, catID, offset, pagesize)
			if err != nil {
				log.Fatalf("категория %d, offset %d: %v", catID, offset, err)
			}
			for _, p := range resp.Items {
				all = append(all, lenta.ProductExport{
					Name:  p.Name,
					Price: float64(p.Prices.Cost) / 100,
					URL:   fmt.Sprintf("https://lenta.com/product/%s-%d/", p.Slug, p.ID),
				})
			}

			offset += pagesize
			if offset >= resp.Total {
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
	}

	if strings.HasSuffix(strings.ToLower(*outputPath), ".csv") {
		if err := lenta.ExportToCSV(all, *outputPath); err != nil {
			log.Fatalf("экспорт CSV: %v", err)
		}
	} else {
		if err := lenta.ExportToJSON(all, *outputPath); err != nil {
			log.Fatalf("эксорт JSON: %v", err)
		}
	}
	log.Printf("Выгружено %d товаров в %s", len(all), *outputPath)
}
