package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"testJob/internal/lenta"
	"time"
)

func main() {
	http.ProxyURL := flag.String("proxy", "", "прокси, например http://user:pass@host:port")
	sessionToken := flag.String("session-token", "", "токкен сессии Lenta (Utk_SessionToken)")
	storeID := flag.Int("store-id", 104, "id магазина")
	domain := flag.String("domain", "moscow", "регион (x-domain) например msocow")
	deviceID := flag.String("device-id", "", "x-device-id (если пусто - генерируется один раз)")
	userSessionID := flag.String("user-session-id", "", "UserSessionId из cookie")
	outputPath := flag.String("out", "lenta_products.json", "файл выгрузки (JSON или .csv)")
	categoryIDStr := flag.String("categories", "2.5", "id категорий через запятую (например 2=Сыры)")
	flag.Parse()

	if *sessionToken == "" {
		log.Fatal("укажите -session-token")
	}
	if *proxyURL == "" {
		log.Fatal()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var all []lenta.ProductExport
	const pagesize = 40

	for _, catID := range cfg.CategoryID {
		offset := 0
		for {
			resp, err := lenta.GetCategoryItems(ctx, client, catID, offset, pagesize)
			if err != nil {
				log.Fatalf("категория %d, offset %d: %v", catID, offset, err)
			}
			for _, p := range resp.Items {
				all = append(all, lenta.ProductExport{
					Name: p.Name,
					Price: float64(p.Price.Cost) / 100,
					URL: fmt.Sprintf("https://lenta.com/product/%s-%d/", p.Slug, p.ID),
				})
			}
			offset += pagesize
			if offset >= resp.Total {
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
	}

	if len(*outputPath) > 4 && (*outPath)[len(*outputPath)-4:] == ".csv" {
		if err := lenta.ExportToCSV(all, *outPath); err != nil {
			log.Fatalf("экспорт CSV: %v", err)
		}
	} else {
		if err := lenta.ExportToJSON(all, *outputPath); err != nil {
			log.Fatalf("эксорт JSON: %v", err)
		}
	}
	log.Printf("Выгружено %d товаров в %s", len(all), *outPath)
}


