CREATE TYPE user_role AS ENUM ('admin', 'viewer');
ALTER TABLE api_keys ADD COLUMN role user_role NOT NULL DEFAULT 'admin';
