-- 1. Create schema
CREATE SCHEMA IF NOT EXISTS task;

-- 2. Create tables

CREATE TABLE task.customers (
    id VARCHAR PRIMARY KEY
);

CREATE TABLE task.orders (
    id VARCHAR PRIMARY KEY,
    customer_id VARCHAR REFERENCES task.customers(id),
    track_number VARCHAR,
    entry VARCHAR,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE task.deliveries (
    id SERIAL PRIMARY KEY,
    order_uid VARCHAR NOT NULL REFERENCES task.orders(id),
    name VARCHAR,
    phone VARCHAR,
    zip VARCHAR,
    city VARCHAR,
    address VARCHAR,
    region VARCHAR,
    email VARCHAR
);

CREATE TABLE task.payments (
    id SERIAL PRIMARY KEY,
    order_uid VARCHAR NOT NULL REFERENCES task.orders(id),
    transaction VARCHAR,
    request_id VARCHAR,
    currency VARCHAR,
    provider VARCHAR,
    amount INTEGER,
    payment_dt BIGINT,
    bank VARCHAR,
    delivery_cost INTEGER,
    goods_total INTEGER,
    custom_fee INTEGER
);

CREATE TABLE task.items (
    id SERIAL PRIMARY KEY,
    order_uid VARCHAR NOT NULL REFERENCES task.orders(id),
    chrt_id BIGINT,
    track_number VARCHAR,
    price INTEGER,
    name VARCHAR,
    sale INTEGER,
    size VARCHAR,
    total_price INTEGER,
    nm_id BIGINT,
    brand VARCHAR,
    status INTEGER
);

-- 3. Create task_user with limited privileges (only if it doesn't exist)
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'task_user') THEN
        CREATE USER task_user WITH PASSWORD 'pass123!!!';
    END IF;
END
$$;

-- 4. Grant schema usage
GRANT USAGE ON SCHEMA task TO task_user;

-- Grant privileges on all existing tables in the schema
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA task TO task_user;
GRANT TRUNCATE ON ALL TABLES IN SCHEMA task TO task_user;

-- Grant privileges on all existing sequences in the schema
GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA task TO task_user;

-- Set default privileges for future tables created by the current user in the schema
ALTER DEFAULT PRIVILEGES IN SCHEMA task
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO task_user;

-- Set default privileges for future sequences created by the current user in the schema
ALTER DEFAULT PRIVILEGES IN SCHEMA task
GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO task_user;
