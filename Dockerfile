FROM golang:1.25-alpine AS builder

WORKDIR /src

# Install git for module resolution when needed.
RUN apk add --no-cache git

# Cache Go modules first for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build the API binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=builder /out/api /app/api

EXPOSE 8080

ENTRYPOINT ["/app/api"]
