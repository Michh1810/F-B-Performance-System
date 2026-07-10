# FBPerformance Backend

This is the backend service for FBPerformance, built with Golang, PostgreSQL, and Redis. 

The entire development environment is Dockerized. You do **not** need to install Go, PostgreSQL, or Redis on your local machine to work on this project!

## Prerequisites

1. Install [Docker](https://docs.docker.com/get-docker/) (Docker Desktop for Mac/Windows).
2. (Optional but recommended) Install `golang-migrate` to run database migrations locally.
   - macOS: `brew install golang-migrate`

## Getting Started

### 1. Start the Environment
To spin up the Go backend, PostgreSQL database, and Redis cache, run:

```bash
docker-compose up -d --build
```
*Note: The Go server uses `air` for hot-reloading. Any changes you make to `.go` files will instantly restart the server in the container!*

### 2. Verify the Server
Open your browser and navigate to:
[http://localhost:8080](http://localhost:8080)

### 3. Run Database Migrations
Before the application can work properly, you need to set up the database tables.

Run the following command from the root directory to apply the latest database schema:
```bash
migrate -path migrations -database "postgres://postgres:devpassword@localhost:5440/fbperformance?sslmode=disable" up
```

## Useful Commands

- **Stop the environment:** `docker-compose down`
- **View backend logs:** `docker logs fbperformance-backend -f`
- **Access PostgreSQL CLI:** `docker exec -it fbperformance-postgres psql -U postgres -d fbperformance`

## Database Connection
If you want to use a GUI like TablePlus or DBeaver to view the database:
- **Host:** `localhost`
- **Port:** `5440`
- **User:** `postgres`
- **Password:** `devpassword`
- **Database:** `fbperformance`
