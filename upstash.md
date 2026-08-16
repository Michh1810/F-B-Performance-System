# Upstash Redis Setup (Free Tier)

Upstash offers a serverless Redis instance with a free tier (up to 256 MB). Follow these steps to provision a Redis instance for the FBPerformance backend:

1. **Create an Upstash account** – go to https://upstash.com and sign up.
2. **Create a new Redis database** – choose the *Free* plan. After creation you will receive a connection URL that looks like:
   ```
   redis://<username>:<password>@<host>:<port>
   ```
3. **Add the URL to environment variables** – copy the URL into the `REDIS_URL` variable in `.env.example` (already added) and into the Render service configuration (`render.yaml`).
4. **Configure the backend** – the Go application reads `REDIS_URL` from the environment and uses it to create a Redis client. No code changes are required.
5. **Optional – test the connection** – you can run a simple Go snippet or use `redis-cli` locally to verify connectivity:
   ```bash
   redis-cli -u $REDIS_URL ping
   ```

The free tier is sufficient for development and low‑traffic production workloads.
