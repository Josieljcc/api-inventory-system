# Inventory System API

[![CI](https://github.com/<your-username>/<your-repo>/actions/workflows/ci.yml/badge.svg)](https://github.com/<your-username>/<your-repo>/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/<your-username>/<your-repo>/branch/main/graph/badge.svg)](https://codecov.io/gh/<your-username>/<your-repo>)

Inventory management system built with Go and PostgreSQL.

## Getting Started

### Run with Docker Compose
```sh
docker-compose up -d
```
The API will be available at http://localhost:8080

### Environment Variables
- `DB_URL`: Database connection string (default: `postgres://user:password@db:5432/inventory?sslmode=disable`)
- `JWT_SECRET`: Secret for signing JWT tokens (default: `changeme`)

### Manual Build
```sh
go build -o inventory-system ./cmd/server
```

## Authentication
All product routes require JWT authentication. Obtain a token via `/register` or `/login` and send it in the header:
```
Authorization: Bearer <token>
```

## API Documentation
Interactive Swagger docs: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)

## Main Endpoints
- `POST   /register` — register a new user
- `POST   /login` — authenticate and get JWT + refresh token
- `POST   /refresh` — get new JWT using refresh token
- `POST   /products` — create product (private)
- `GET    /products` — list products (private)
- `GET    /products/{barcode}` — get product by barcode (private)
- `PUT    /products/{id}` — update product (private)
- `DELETE /products/{id}` — delete product (private)
- `POST   /products/{barcode}/entry` — stock entry (private)
- `POST   /products/{barcode}/exit` — stock exit (private)

## Example Usage (curl)
### Register
```sh
curl -X POST http://localhost:8080/register -H 'Content-Type: application/json' -d '{"username":"johndoe","password":"secret"}'
```
### Login
```sh
curl -X POST http://localhost:8080/login -H 'Content-Type: application/json' -d '{"username":"johndoe","password":"secret"}'
```
### Refresh Token
```sh
curl -X POST http://localhost:8080/refresh -H 'Content-Type: application/json' -d '{"refresh_token":"<refresh>"}'
```
### Create Product
```sh
curl -X POST http://localhost:8080/products -H 'Content-Type: application/json' -H 'Authorization: Bearer <token>' -d '{"name":"Apple","barcode":"123456","quantity":10,"min_stock":2}'
```

## Running Tests
```sh
go test ./...
```

---

For more details, see the Swagger documentation or the codebase. 