CREATE TABLE IF NOT EXISTS tutor_ratings (
    id bigserial PRIMARY KEY,
    tutor_id VARCHAR NOT NULL REFERENCES tutors(ivw_id) ON DELETE CASCADE,
    rating INTEGER NOT NULL,
    count INTEGER NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tutor_ratings_tutor_id ON tutor_ratings(tutor_id);
