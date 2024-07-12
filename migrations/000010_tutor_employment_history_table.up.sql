CREATE TABLE IF NOT EXISTS tutor_employment_history (
    id bigserial PRIMARY KEY,
    tutor_id VARCHAR NOT NULL REFERENCES tutors(ivw_id) ON DELETE CASCADE,
    company VARCHAR(255) NOT NULL,
    position VARCHAR(255) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tutor_employment_history_tutor_id ON tutor_employment_history(tutor_id);
