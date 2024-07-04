ALTER TABLE users ADD CONSTRAINT check_role CHECK (role IN ('admin', 'tutor', 'student'));
ALTER TABLE users ADD CONSTRAINT check_gender CHECK (gender IN ('male', 'female', 'prefer not to say'));
ALTER TABLE users ADD CONSTRAINT check_criminal_record CHECK (criminal_record IN (TRUE, FALSE));
ALTER TABLE users ADD CONSTRAINT check_eligible_to_work CHECK (eligible_to_work IN (TRUE, FALSE));
ALTER TABLE users ADD CONSTRAINT check_zipcode CHECK (zipcode ~ '^[0-9]+$');
ALTER TABLE users ADD CONSTRAINT check_date_of_birth CHECK (date_of_birth < NOW());
