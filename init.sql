-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create events table
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    duration_hours INT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slots JSONB NOT NULL DEFAULT '[]', -- Using JSONB to store the slots as a list of objects with start_time and end_time instead of normalizing the table for better performance and easier maintenance.
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create users availability table
CREATE TABLE IF NOT EXISTS users_availability (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, start_time, end_time)
);