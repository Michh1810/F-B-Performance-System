# Database Migrations

This folder contains the database migration files. 

For Golang, a very common and robust tool for handling migrations is [golang-migrate/migrate](https://github.com/golang-migrate/migrate).

## Getting Started with `golang-migrate`

### 1. Install the CLI tool
On macOS, you can install it using Homebrew:
```bash
brew install golang-migrate
```

### 2. Create a new migration
To create your first migration (e.g., creating a users table):
```bash
migrate create -ext sql -dir migrations -seq create_users_table
```
This will generate two files in this folder:
- `000001_create_users_table.up.sql`
- `000001_create_users_table.down.sql`

### 3. Run migrations against your local PostgreSQL database
Based on `docker-compose.yml`, you can run migrations using the following command:

```bash
migrate -path migrations -database "postgres://postgres:devpassword@localhost:5432/fbperformance?sslmode=disable" up
```

### 4. Rollback migrations
To undo the last migration:
```bash
migrate -path migrations -database "postgres://postgres:devpassword@localhost:5440/fbperformance?sslmode=disable" down 1
```
