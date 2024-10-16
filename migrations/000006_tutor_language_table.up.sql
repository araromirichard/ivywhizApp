CREATE TABLE
IF NOT EXISTS tutor_languages
(
    id bigserial PRIMARY KEY,
    tutor_id VARCHAR UNIQUE NOT NULL REFERENCES tutors
(ivw_id) ON
DELETE CASCADE,
    languages text[]
NOT NULL DEFAULT '{}',
    created_at timestamp
(0)
with time zone NOT NULL DEFAULT NOW
(),
    updated_at timestamp
(0)
with time zone NOT NULL DEFAULT NOW
()
);

-- Create an index on tutor_id
CREATE INDEX idx_tutor_languages_tutor_id ON tutor_languages(tutor_id);
