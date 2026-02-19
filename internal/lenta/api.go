package lenta

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// FetchCategory выполняет POST-запрос к catalog API.
//
// Требования:
// - Валидный sessiontoken
// - Anti-bot cookies
// - Корректный TLS fingerprint (uTLS)
// Возвращает десериализованный JSON ответ.

func FetchCategory(client *Client, categoryID int, offset int, limit int) (*CatalogItemsResponse, error) {
	urlStr := client.BaseURL() + "/api-gateway/v1/catalog/items"

	payload := map[string]interface{}{
		"categoryId": categoryID,
		"filters": map[string]interface{}{
			"range":         []interface{}{},
			"checkbox":      []interface{}{},
			"multicheckbox": []interface{}{},
		},
		"limit":  limit,
		"offset": offset,
		"sort": map[string]string{
			"type":  "popular",
			"order": "desc",
		},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", urlStr, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	log.Printf("POST к %s с телом: %s", urlStr, string(bodyBytes))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ошибка API: %s. Тело: %s", resp.Status, string(body))
	}

	var data CatalogItemsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return &data, nil
}
