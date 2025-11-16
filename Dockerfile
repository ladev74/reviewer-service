FROM golang:1.25.1-alpine AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/reviewer-service cmd/reviewer-service/main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/migrate  cmd/migrate/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /out/reviewer-service /app/reviewer-service
COPY --from=builder /out/migrate /app/migrate

COPY config /app/config
COPY database/migrations /app/database/migrations

