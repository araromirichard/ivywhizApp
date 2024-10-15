-- Create the guardians table
CREATE TABLE IF NOT EXISTS guardians
(
    id bigserial PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    student_id VARCHAR NOT NULL REFERENCES students(ivw_id) ON DELETE CASCADE,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    relationship_to_student VARCHAR(50) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    email citext NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    version INT NOT NULL DEFAULT 1
);

-- Index for querying guardians by student ID
CREATE INDEX idx_guardians_student_id ON guardians(student_id);
CREATE INDEX idx_guardians_email ON guardians(email);
