CREATE TABLE IF NOT EXISTS users (
id           SERIAL PRIMARY KEY,
telegram_id  BIGINT UNIQUE NOT NULL,
username     VARCHAR(100),
first_name   VARCHAR(100),
phone        VARCHAR(20),
email        VARCHAR(100),
role         VARCHAR(20) DEFAULT 'user',
created_at   TIMESTAMP DEFAULT NOW()
);