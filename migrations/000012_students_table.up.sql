-- Create an ENUM type for education levels
CREATE TYPE education_level_enum AS ENUM ('preschool', 'primary', 'secondary', 'tertiary', 'other');

-- Create the students table without parent/guardian details
CREATE TABLE IF NOT EXISTS students
(
    id bigserial PRIMARY KEY,
    ivw_id VARCHAR(50) UNIQUE NOT NULL,
    user_id BIGINT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_background TEXT,
    education_level education_level_enum Not NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    version INT NOT NULL DEFAULT 1
);

-- Index for querying students by user ID
CREATE INDEX idx_students_user_id ON students(user_id);
