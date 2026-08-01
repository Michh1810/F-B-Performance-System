# Neon PostgreSQL Setup (Free Tier)

Neon provides a fully‑managed PostgreSQL instance with a generous free tier (up to 10 GB storage, 20 M rows). Follow these steps to provision a Neon database for the FBPerformance backend:

1. **Create a Neon account** – go to https://neon.tech and sign up.
2. **Create a new project** – choose the *Free* plan. Neon will give you a connection string that looks like:
   ```
   postgres://<user>:<password>@<host>/<database>?sslmode=require
   ```
3. **Enable the `pgvector` extension** – Neon supports extensions. In the Neon console, run:
   ```sql
   CREATE EXTENSION IF NOT EXISTS vector;
   ```
   This is required because the project uses the `pgvector/pgvector` Docker image locally.
4. **Run migrations** – after the database is created, set the `DATABASE_URL` environment variable (in Render and locally) to the Neon connection string and execute the migration command:
   ```bash
   migrate -path migrations -database "$DATABASE_URL" up
   ```
5. **Update `.env.example`** – the file already contains a placeholder for `DATABASE_URL`. Replace the local Docker URL with the Neon URL when deploying.

The Neon connection string should be added to the Render service configuration (`render.yaml`) and to the Vercel environment variables if the frontend needs to call the backend directly.
