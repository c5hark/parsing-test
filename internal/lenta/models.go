package lenta

// Модели соответствуют JSON-ответу catalog API.
// Структура может измениться при обновлении frontend-а сайта.
// Поля добавляются по мере необходимости.

type CatalogItemsResponse struct {
	Categories []Category `json:"categories,omitempty"`
	Filters    Filters    `json:"filters,omitempty"`
	Items      []Product  `json:"items"`
	// Total может быть где-то в meta или отдельно — добавь если увидишь в полном ответе
}

type Category struct {
	ID          int    `json:"id"`
	Name        string `json:"name,omitempty"`
	Slug        string `json:"slug,omitempty"`
	HasChildren bool   `json:"hasChildren"`
	// ...
}

type Filters struct {
	Checkbox []interface{} `json:"checkbox"`
	// ...
}

type Product struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Slug     string   `json:"slug"`
	StoreID  int      `json:"storeId"`
	Prices   Prices   `json:"prices"`
	Rating   Rating   `json:"rating,omitempty"`
	Badges   Badges   `json:"badges,omitempty"`
	Features Features `json:"features,omitempty"`
	Weight   Weight   `json:"weight,omitempty"`
	// Добавляй другие поля по мере необходимости
}

type Prices struct {
	Price              int  `json:"price"` // основная цена (со скидкой)
	PriceRegular       int  `json:"priceRegular"`
	Cost               int  `json:"cost"`
	CostRegular        int  `json:"costRegular"`
	IsLoyaltyCardPrice bool `json:"isLoyaltyCardPrice"`
	// ...
}

type Rating struct {
	Rate  float64 `json:"rate"`
	Votes int     `json:"votes"`
}

type Badges struct {
	Discount []DiscountBadge `json:"discount,omitempty"`
	// ...
}

type DiscountBadge struct {
	Title string `json:"title"` // "-23%"
	// ...
}

type Features struct {
	IsAdult   bool   `json:"isAdult"`
	IsAlcohol bool   `json:"isAlcohol"`
	MarkType  string `json:"markType,omitempty"` // "MILK"
	// ...
}

type Weight struct {
	Gross   int    `json:"gross"`
	Package string `json:"package"` // "900мл"
	// ...
}

type ProductExport struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	URL   string  `json:"url"`
}
