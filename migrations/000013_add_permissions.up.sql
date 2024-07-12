-- Create the permissions table if not exists
CREATE TABLE IF NOT EXISTS permissions (
    id bigserial PRIMARY KEY,
    code text NOT NULL
);

-- Insert permissions for admin, tutor, and student
INSERT INTO permissions (code) VALUES
    ('admin:access'),
    ('tutor:access'),
    ('student:access');

-- Create the users_permissions table if not exists
CREATE TABLE IF NOT EXISTS users_permissions (
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    permission_id bigint NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, permission_id)
);
