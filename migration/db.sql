-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY NOT NULL,
    user_id BIGINT UNIQUE NOT NULL,
    name VARCHAR(30),
    rate INT,
    status INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS channels (
    name VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS answers (
    answers TEXT
);

-- Create admins table
CREATE TABLE IF NOT EXISTS admins (
    id BIGINT UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS files (
    id SERIAL PRIMARY KEY,
    file_id TEXT UNIQUE NOT NULL,
    file_name TEXT,
    mime_type TEXT,
    file_data BYTEA,
    created_at TIMESTAMP DEFAULT NOW()
);

