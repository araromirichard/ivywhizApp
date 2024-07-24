CREATE TABLE IF NOT EXISTS tutor_skills (
    id bigserial PRIMARY KEY,
    tutor_id VARCHAR UNIQUE NOT NULL REFERENCES tutors(ivw_id) ON DELETE CASCADE,
    skills text[] NOT NULL DEFAULT '{}',
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
    
);

