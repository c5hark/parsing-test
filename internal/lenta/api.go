package lenta

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type categoriesResponse struct {
	Categories []Category `json:"categories"`
}

func GetCategories(ctx context.Context, c *Client) ([]Category, error) {
	url := c.BaseURL() + "/api.gateway/v1/catalog/categories?timestamp=" + fmt.Sprintf("%d", time.Now().UnixMilli())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad status code: %d (%s)", resp.StatusCode, string(body))
	}
	var out categoriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Categories, nil
}

func GetCategoryItems(ctx context.Context, c *Client, categoryID, offset, limit int) (*CatalogItemsResponse, error) {
	payload := map[string]interface{}{
		"categoryId": categoryID,
		"filters": map[string]interface{}{
			"range":         []interface{}{},
			"checkbox":      []interface{}{},
			"multicheckbox": []interface{}{},
		},
		"limit":  limit,
		"offset": offset,
		"sort": map[string]interface{}{
			"type":  "popular",
			"order": "desc",
		},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := c.BaseURL() + "/api-gateway/v1/catalog/items"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("catalog/items: status %d, body: %s", resp.StatusCode, string(b))
	}

	var out CatalogItemsResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
