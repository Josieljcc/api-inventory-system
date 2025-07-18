package products

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	DB *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) CreateProduct(ctx context.Context, p *Product) error {
	query := `INSERT INTO products (name, barcode, quantity, min_stock) VALUES ($1, $2, $3, $4) RETURNING id`
	return r.DB.QueryRow(ctx, query, p.Name, p.Barcode, p.Quantity, p.MinStock).Scan(&p.ID)
}

func (r *Repository) GetProducts(ctx context.Context, q ProductsQuery) ([]Product, int, error) {
	args := []interface{}{}
	where := ""
	idx := 1
	if q.Name != "" {
		where += " AND name ILIKE $" + strconv.Itoa(idx)
		args = append(args, "%"+q.Name+"%")
		idx++
	}
	if q.Barcode != "" {
		where += " AND barcode = $" + strconv.Itoa(idx)
		args = append(args, q.Barcode)
		idx++
	}
	if q.MinStock > 0 {
		where += " AND min_stock >= $" + strconv.Itoa(idx)
		args = append(args, q.MinStock)
		idx++
	}
	orderBy := "id"
	if q.Sort == "name" || q.Sort == "quantity" || q.Sort == "min_stock" {
		orderBy = q.Sort
	}
	order := "ASC"
	if q.Order == "desc" {
		order = "DESC"
	}
	limit := q.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (q.Page - 1) * limit
	query := "SELECT id, name, barcode, quantity, min_stock FROM products WHERE 1=1" + where + " ORDER BY " + orderBy + " " + order + " LIMIT $" + strconv.Itoa(idx) + " OFFSET $" + strconv.Itoa(idx+1)
	args = append(args, limit, offset)
	rows, err := r.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Barcode, &p.Quantity, &p.MinStock); err != nil {
			return nil, 0, err
		}
		products = append(products, p)
	}
	// Total count
	total := 0
	countQuery := "SELECT COUNT(*) FROM products WHERE 1=1" + where
	if err := r.DB.QueryRow(ctx, countQuery, args[:idx-1]...).Scan(&total); err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

func (r *Repository) GetProductByBarcode(ctx context.Context, barcode string) (*Product, error) {
	var p Product
	err := r.DB.QueryRow(ctx, `SELECT id, name, barcode, quantity, min_stock FROM products WHERE barcode=$1`, barcode).Scan(&p.ID, &p.Name, &p.Barcode, &p.Quantity, &p.MinStock)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) UpdateProduct(ctx context.Context, id int, p *Product) error {
	cmd, err := r.DB.Exec(ctx, `UPDATE products SET name=$1, barcode=$2, quantity=$3, min_stock=$4 WHERE id=$5`, p.Name, p.Barcode, p.Quantity, p.MinStock, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("product not found")
	}
	return nil
}

func (r *Repository) DeleteProduct(ctx context.Context, id int) error {
	cmd, err := r.DB.Exec(ctx, `DELETE FROM products WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("product not found")
	}
	return nil
}

func (r *Repository) StockEntry(ctx context.Context, barcode string, qty int) error {
	cmd, err := r.DB.Exec(ctx, `UPDATE products SET quantity = quantity + $1 WHERE barcode = $2`, qty, barcode)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("product not found")
	}
	return nil
}

func (r *Repository) StockExit(ctx context.Context, barcode string, qty int) error {
	cmd, err := r.DB.Exec(ctx, `UPDATE products SET quantity = quantity - $1 WHERE barcode = $2 AND quantity >= $1`, qty, barcode)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("Insufficient stock or product not found")
	}
	return nil
}

type RepositoryInterface interface {
	CreateProduct(ctx context.Context, p *Product) error
	GetProducts(ctx context.Context, q ProductsQuery) ([]Product, int, error)
	GetProductByBarcode(ctx context.Context, barcode string) (*Product, error)
	UpdateProduct(ctx context.Context, id int, p *Product) error
	DeleteProduct(ctx context.Context, id int) error
	StockEntry(ctx context.Context, barcode string, qty int) error
	StockExit(ctx context.Context, barcode string, qty int) error
}
