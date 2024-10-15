-- Drop foreign key constraints that depend on the users table first
ALTER TABLE addresses DROP CONSTRAINT IF EXISTS addresses_user_id_fkey;

-- Drop other foreign key constraints if necessary (e.g., from other tables)
-- For example, if there are any other tables referencing users, drop them here.

-- Drop check constraints from the users table
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_role;
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_gender;
ALTER TABLE users DROP CONSTRAINT IF EXISTS check_date_of_birth;
