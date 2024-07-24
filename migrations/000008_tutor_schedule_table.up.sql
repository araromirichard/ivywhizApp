CREATE TABLE IF NOT EXISTS tutor_schedule (
    id bigserial PRIMARY KEY,
    tutor_id VARCHAR NOT NULL REFERENCES tutors(ivw_id) ON DELETE CASCADE,
    day VARCHAR(50) NOT NULL,
    start_time time NOT NULL,
    end_time time NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tutor_schedule_tutor_id ON tutor_schedule(tutor_id);
