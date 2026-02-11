# events-system

Simple Go HTTP service with a PostgreSQL-backed events and users system for scheduling events and finding optimal meeting times.

## Requirements

- Docker + Docker Compose

## Database schema

The schema lives in `init.sql` and creates:

- `users` table: stores user information
- `events` table: stores events with JSONB slots
- `users_availability` table: stores user availability slots

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

## API Examples

All timestamps are Unix epoch seconds (int64). The API accepts and returns times as integers.

### 1. Create Users

Create two users:

```bash
# Create User 1 (Alice)
curl -X POST "http://localhost:8080/api/users" \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com"}'

# Create User 2 (Bob)
curl -X POST "http://localhost:8080/api/users" \
  -H "Content-Type: application/json" \
  -d '{"name":"Bob","email":"bob@example.com"}'
```

**Note:** Copy the `id` from each response for the examples below. Replace `$USER1_ID` and `$USER2_ID` in subsequent commands.

### 2. Create an Event

Create an event with 3-4 slots (1 hour duration each). Each slot represents a 1-hour window:

```bash
# Slot 1: 09:00-10:00 UTC (start_time: 1770800400, end_time: 1770804000)
# Slot 2: 11:00-12:00 UTC (start_time: 1770807600, end_time: 1770811200)
# Slot 3: 14:00-15:00 UTC (start_time: 1770818400, end_time: 1770822000)
# Slot 4: 16:00-17:00 UTC (start_time: 1770825600, end_time: 1770829200)

curl -X POST "http://localhost:8080/api/events" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Team Meeting",
    "duration_hours": 1,
    "organizer_id": "'$USER1_ID'",
    "slots": [
      {"start_time": 1770800400, "end_time": 1770804000},
      {"start_time": 1770807600, "end_time": 1770811200},
      {"start_time": 1770818400, "end_time": 1770822000},
      {"start_time": 1770825600, "end_time": 1770829200}
    ]
  }'
```

**Note:**

- Replace timestamps with actual Unix epoch seconds for your desired date/time
- Each slot is 1 hour (3600 seconds difference between start and end)
- To get current timestamps: `date +%s` (current time) or calculate for future dates

### 3. Add User Availability Slots

Add 2-3 availability slots per user. At least one slot per user should overlap with an event slot.

```bash
# Add slots for User 1 (Alice) - Wednesday, 11 February 2026
# Slot 1: 08:50-10:10 (overlaps with event slot 1: 09:00-10:00) - 80 minutes
# Slot 2: 13:50-15:10 (overlaps with event slot 3: 14:00-15:00) - 80 minutes
# Slot 3: 07:00-07:50 (doesn't overlap) - 50 minutes

curl -X POST "http://localhost:8080/api/users/$USER1_ID/slots" \
  -H "Content-Type: application/json" \
  -d '[
    {"start_time": 1770799800, "end_time": 1770804600},
    {"start_time": 1770817800, "end_time": 1770822600},
    {"start_time": 1770793200, "end_time": 1770796200}
  ]'

# Add slots for User 2 (Bob) - Wednesday, 11 February 2026
# Slot 1: 09:00-10:00 (exactly matches event slot 1) - 60 minutes
# Slot 2: 11:00-12:00 (exactly matches event slot 2) - 60 minutes
# Slot 3: 18:00-18:30 (doesn't overlap) - 30 minutes

curl -X POST "http://localhost:8080/api/users/$USER2_ID/slots" \
  -H "Content-Type: application/json" \
  -d '[
    {"start_time": 1770800400, "end_time": 1770804000},
    {"start_time": 1770807600, "end_time": 1770811200},
    {"start_time": 1770832800, "end_time": 1770834600}
  ]'
```

**Note:**

- User slots can be smaller than 1 hour (as shown above: 30, 50, 60, 80 minutes)
- At least one slot per user overlaps with an event slot:
  - User 1 overlaps with event slots 1 and 3
  - User 2 overlaps with event slots 1 and 2
- The best possible slot should return event slot 1 (09:00-10:00) with 2 users available

### 4. Get Best Possible Event Slot

Find the event slot with maximum user attendance:

```bash
# Replace EVENT_ID with the ID returned from the create event request
curl -X GET "http://localhost:8080/api/events/$EVENT_ID/possible-slot" \
  -H "Content-Type: application/json"
```

This will return the event slot with the most available users (should show 2 users in this example).

## API Endpoints

- **Health**: `GET /api/health`
- **Create user**: `POST /api/users`
- **Get user**: `GET /api/users/{id}`
- **Get all users**: `GET /api/users`
- **Create user slots**: `POST /api/users/{id}/slots`
- **Delete user slots**: `DELETE /api/users/{id}/slots`
- **Create event**: `POST /api/events`
- **Get event**: `GET /api/events/{id}`
- **Update event**: `PUT /api/events/{id}`
- **Delete event**: `DELETE /api/events/{id}`
- **Get possible event slot**: `GET /api/events/{id}/possible-slot`

## Calculating Timestamps

To generate Unix epoch timestamps for your dates, use:

- https://www.epochconverter.com/
- Enter your date/time and get the Unix timestamp
