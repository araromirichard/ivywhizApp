ALTER TABLE users ADD CONSTRAINT check_role CHECK (role IN ('admin', 'tutor', 'student'));
ALTER TABLE users ADD CONSTRAINT check_gender CHECK (gender IN ('male', 'female', 'prefer not to say'));
ALTER TABLE users ADD CONSTRAINT check_date_of_birth CHECK (date_of_birth < NOW());
