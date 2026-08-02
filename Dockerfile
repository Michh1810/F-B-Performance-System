FROM golang:1.25-alpine AS base
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download

FROM base AS dev
RUN go install github.com/air-verse/air@latest
COPY . .
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

FROM base AS builder
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/api ./cmd/api

FROM alpine:3.21 AS prod
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/bin/api /app/api
COPY migrations /app/migrations
EXPOSE 8080
CMD ["/app/api"]
