package products

type Product struct {
	ID       int    `json:"id"`
	Name     string `json:"name" validate:"required"`
	Barcode  string `json:"barcode" validate:"required"`
	Quantity int    `json:"quantity"`
	MinStock int    `json:"min_stock"`
}

type StockRequest struct {
	Quantity int `json:"quantity" validate:"required,gte=1"`
}
