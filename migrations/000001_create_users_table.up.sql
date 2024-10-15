-- Enable citext extension for case-insensitive text fields
CREATE EXTENSION IF NOT EXISTS citext;

-- Create the users table
CREATE TABLE IF NOT EXISTS users
(
    id bigserial PRIMARY KEY,
    email citext UNIQUE NOT NULL,
    password bytea NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    username VARCHAR(50),
    role VARCHAR(20) DEFAULT 'student',
    about_yourself text,
    date_of_birth DATE,
    gender VARCHAR(10),
    activated bool NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    version integer NOT NULL DEFAULT 1
);

-- Create indexes for performance optimization
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
