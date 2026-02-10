package lenta

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	HasChildren bool   `json:"hasChildren"`
	ParentID    int    `json:"parentId"`
	ParentName  string `json:"parentName"`
	Level       int    `json:"level"`
}

type CatalogItemsRequest struct {
	CategoryID int      `json:"categoryId"`
	Filters    Filters  `json:"filters"`
	Limit      int      `json:"limit"`
	Offset     int      `json:"offset"`
	Sort       SortOpts `json:"sort"`
}

type Filters struct {
	Range         []interface{} `json:"range"`
	Checkbox      []interface{} `json:"checkbox"`
	Multicheckbox []interface{} `json:"multicheckbox"`
}

type SortOpts struct {
	Type  string `json:"type"`
	Order string `json:"order"`
}

type CatalogItemsResponse struct {
	Total int       `json:"total"`
	Items []Product `json:"items"`
}

type Product struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	StoreID int    `json:"storeId"`
	Prices  Prices `json:"prices"`
}

type Prices struct {
	Cost        int `json:"cost"`
	Price       int `json:"price"`
	CostRegular int `json:"costRegular"`
}

type ProductExport struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	URL   string  `json:"url"`
}
