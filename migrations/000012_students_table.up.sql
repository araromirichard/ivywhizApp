-- Check if the ENUM type already exists, if not, create it
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'education_level_enum') THEN
        CREATE TYPE education_level_enum AS ENUM ('preschool', 'primary', 'secondary', 'tertiary');
    END IF;
END$$;

-- Create the students table without parent/guardian details
CREATE TABLE IF NOT EXISTS students
(
    id bigserial PRIMARY KEY,
    ivw_id VARCHAR(50) UNIQUE NOT NULL,
    user_id BIGINT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_background TEXT,
    education_level education_level_enum NOT NULL,
    created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
    version INT NOT NULL DEFAULT 1
);

-- Index for querying students by user_id
CREATE INDEX IF NOT EXISTS idx_students_user_id ON students(user_id);
