CREATE TABLE IF NOT EXISTS students (
    id bigserial PRIMARY KEY,
    user_id BIGINT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    family_background TEXT,
    parent_first_name VARCHAR(50) NOT NULL,
    parent_last_name VARCHAR(50) NOT NULL,
    parent_relationship_to_child VARCHAR(50) NOT NULL,
    parent_phone VARCHAR(20) NOT NULL,
    parent_email citext NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_students_user_id ON students(user_id);
CREATE INDEX idx_students_parent_email ON students(parent_email);
