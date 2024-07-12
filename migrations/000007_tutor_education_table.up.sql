CREATE TABLE IF NOT EXISTS tutor_education (
    id bigserial PRIMARY KEY,
    tutor_id VARCHAR NOT NULL REFERENCES tutors(ivw_id) ON DELETE CASCADE,
    course VARCHAR(255) NOT NULL,
    study_period VARCHAR(50) NOT NULL,
    institute VARCHAR(255) NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tutor_education_tutor_id ON tutor_education(tutor_id);
