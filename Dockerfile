# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
# RUN go mod tidy (removido para evitar erro)
COPY . .
RUN go build -o inventory-system ./cmd/server

FROM alpine:latest AS prod
WORKDIR /root/
COPY --from=builder /app/inventory-system .
COPY internal/database/migrations.sql internal/database/migrations.sql
COPY docs/ ./docs/
EXPOSE 8080
ENV DB_URL="postgres://user:password@db:5432/inventory?sslmode=disable"
CMD ["./inventory-system"]

FROM builder AS dev
CMD ["sleep", "infinity"]

FROM builder AS debug
CMD ["/bin/sh"] 