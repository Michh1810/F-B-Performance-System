# Use the official Golang image as a development environment
FROM golang:alpine

# Set the working directory inside the container
WORKDIR /app

# Install system dependencies
RUN apk update && apk add --no-cache git

# Install 'air' for hot-reloading in development
RUN go install github.com/air-verse/air@latest

# Pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod ./
# COPY go.sum ./ (Uncomment when you have a go.sum)
RUN go mod download && go mod verify

# Copy the rest of the application source code
COPY . .

# Expose port 8080 for the Go application
EXPOSE 8080

# Run 'air' for hot-reloading
CMD ["air", "-c", ".air.toml"]
