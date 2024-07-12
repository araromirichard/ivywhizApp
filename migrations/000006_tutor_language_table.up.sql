CREATE TABLE IF NOT EXISTS tutor_languages (
    id bigserial PRIMARY KEY,
    tutor_id VARCHAR NOT NULL REFERENCES tutors(ivw_id) ON DELETE CASCADE,
    language VARCHAR(50) NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tutor_languages_tutor_id ON tutor_languages(tutor_id);
