package products

type ProductRequest struct {
	Name     string `json:"name"`
	Barcode  string `json:"barcode"`
	Quantity int    `json:"quantity"`
	MinStock int    `json:"min_stock"`
}

type ProductResponse struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Barcode  string `json:"barcode"`
	Quantity int    `json:"quantity"`
	MinStock int    `json:"min_stock"`
}
