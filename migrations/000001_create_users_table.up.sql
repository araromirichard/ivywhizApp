CREATE TABLE IF NOT EXISTS users (
    id bigserial PRIMARY KEY,
    email citext UNIQUE NOT NULL,
    password bytea NOT NULL,
    first_name VARCHAR(50),
    last_name VARCHAR(50),
    activated bool NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    updated_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    role VARCHAR(20) DEFAULT 'student',
    country VARCHAR(100),
    user_photo VARCHAR(255),
    about_yourself text,
    date_of_birth DATE,
    gender VARCHAR(10),
    street_address_1 VARCHAR(255),
    street_address_2 VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(100),
    zipcode VARCHAR(20),
    timezone VARCHAR(50),
    criminal_record BOOLEAN DEFAULT FALSE,
    eligible_to_work BOOLEAN DEFAULT FALSE,
    class_preferences text[],
    version integer NOT NULL DEFAULT 1
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
