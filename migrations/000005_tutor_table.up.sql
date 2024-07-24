CREATE TABLE IF NOT EXISTS tutors (
    id bigserial PRIMARY KEY,
    ivw_id VARCHAR(50) UNIQUE NOT NULL,
    user_id BIGINT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    verification BOOLEAN NOT NULL,
    rate_per_hour DECIMAL(10, 2) NOT NULL,
    eligible_to_work BOOLEAN NOT NULL,
    criminal_record BOOLEAN NOT NULL,
    timezone VARCHAR(50) NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    version INT NOT NULL DEFAULT 1
);

CREATE INDEX idx_tutors_user_id ON tutors(user_id);
