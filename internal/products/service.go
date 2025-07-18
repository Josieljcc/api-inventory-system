package products

import (
	"context"
)

type Service struct {
	Repo RepositoryInterface
}

func NewService(repo RepositoryInterface) *Service {
	return &Service{Repo: repo}
}

func (s *Service) CreateProduct(ctx context.Context, p *Product) error {
	return s.Repo.CreateProduct(ctx, p)
}

type ProductsQuery struct {
	Page     int
	Limit    int
	Name     string
	Barcode  string
	MinStock int
	Sort     string
	Order    string
}

func (s *Service) GetProducts(ctx context.Context, q ProductsQuery) ([]Product, int, error) {
	return s.Repo.GetProducts(ctx, q)
}

func (s *Service) GetProductByBarcode(ctx context.Context, barcode string) (*Product, error) {
	return s.Repo.GetProductByBarcode(ctx, barcode)
}

func (s *Service) UpdateProduct(ctx context.Context, id int, p *Product) error {
	return s.Repo.UpdateProduct(ctx, id, p)
}

func (s *Service) DeleteProduct(ctx context.Context, id int) error {
	return s.Repo.DeleteProduct(ctx, id)
}

func (s *Service) StockEntry(ctx context.Context, barcode string, qty int) error {
	return s.Repo.StockEntry(ctx, barcode, qty)
}

func (s *Service) StockExit(ctx context.Context, barcode string, qty int) error {
	return s.Repo.StockExit(ctx, barcode, qty)
}
