package products

import (
	"context"
	"inventory-system/internal/notifications"
)

type Service struct {
	Repo     RepositoryInterface
	Notifier *notifications.NotificationService
}

func NewService(repo RepositoryInterface, notifier *notifications.NotificationService) *Service {
	return &Service{Repo: repo, Notifier: notifier}
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
	err := s.Repo.StockExit(ctx, barcode, qty)
	if err != nil {
		return err
	}
	p, _ := s.Repo.GetProductByBarcode(ctx, barcode)
	if p != nil && p.Quantity < p.MinStock && s.Notifier != nil {
		s.Notifier.Notify(notifications.NotificationEvent{
			Type:    "low_stock",
			To:      "5586998277053",
			Message: "Product '" + p.Name + "' is below minimum stock!",
			Data:    map[string]interface{}{"barcode": p.Barcode, "quantity": p.Quantity, "min_stock": p.MinStock},
		})
	}
	return nil
}
