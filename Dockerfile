# ---------- Base ----------
FROM golang:1.26-alpine AS base

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

# ---------- Development ----------
FROM base AS dev

RUN go install github.com/air-verse/air@latest

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]

# ---------- Builder ----------
FROM base AS builder

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" \
    -o /out/api ./cmd/api

# ---------- Production ----------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /out/api /app/api

EXPOSE 8080

ENTRYPOINT ["/app/api"]