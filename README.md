# events-system

Simple Go HTTP service with a PostgreSQL-backed `users` table.

## Requirements

- Docker + Docker Compose

## Database schema

The schema lives in `init.sql` and creates the `users` table:

- `user_id` (UUID, primary key)
- `user_name` (string)
- `user_email` (string, unique)

## Getting Started

### Start the application

```bash
docker compose up --build
```

This will:

- Build the application Docker image
- Start PostgreSQL database
- Start the application server
- Automatically run `init.sql` on first startup to create the database schema

The application will be available at `http://localhost:8080`

### Run in background

```bash
docker compose up -d --build
```

### View logs

```bash
# View all logs
docker compose logs -f

# View app logs only
docker compose logs -f app

# View postgres logs only
docker compose logs -f postgres
```

### Stop the application

```bash
docker compose down
```

### Reset database (removes all data)

If you need to re-run `init.sql` or reset the database:

```bash
docker compose down -v
docker compose up --build
```

## API

- **Health**: `GET /api/health`
- **Create user**: `POST /api/users`

Example:

```bash
curl -X POST "http://localhost:8080/api/v1/users" \
  -H "Content-Type: application/json" \
  -d '{"name":"Pulkit","email":"pulkit@example.com"}'
```
