BEGIN;

CREATE SCHEMA IF NOT EXISTS audit;
CREATE SCHEMA IF NOT EXISTS raw;
CREATE SCHEMA IF NOT EXISTS core;
CREATE SCHEMA IF NOT EXISTS mart;
CREATE SCHEMA IF NOT EXISTS quality;
CREATE SCHEMA IF NOT EXISTS benchmark;

CREATE TABLE IF NOT EXISTS audit.source_file (
    source_file_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_name text NOT NULL UNIQUE,
    source_url text,
    source_sha256 text,
    source_period date,
    source_half text CHECK (source_half IN ('first', 'second') OR source_half IS NULL),
    retrieved_at timestamptz,
    last_modified text,
    etag text,
    byte_count bigint CHECK (byte_count >= 0),
    raw_row_count bigint NOT NULL DEFAULT 0 CHECK (raw_row_count >= 0),
    accepted_row_count bigint NOT NULL DEFAULT 0 CHECK (accepted_row_count >= 0),
    rejected_row_count bigint NOT NULL DEFAULT 0 CHECK (rejected_row_count >= 0),
    provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (source_sha256 IS NULL OR source_sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (source_url IS NULL OR source_url ~ '^https://')
);

CREATE UNLOGGED TABLE IF NOT EXISTS raw.import_buffer (
    settlement_date text,
    cusip text,
    symbol text,
    quantity_fails text,
    description text,
    price text,
    source_file text,
    source_row text
);

CREATE TABLE IF NOT EXISTS raw.ftd_observation (
    source_file_id bigint NOT NULL REFERENCES audit.source_file(source_file_id),
    source_row bigint NOT NULL CHECK (source_row > 1),
    settlement_date_text text NOT NULL,
    cusip_text text NOT NULL,
    symbol_text text NOT NULL,
    quantity_fails_text text NOT NULL,
    description_text text NOT NULL,
    price_text text,
    loaded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (source_file_id, source_row)
);

CREATE TABLE IF NOT EXISTS core.ftd_observation (
    settlement_date date NOT NULL,
    cusip varchar(9) NOT NULL,
    symbol varchar(32) NOT NULL,
    quantity_fails bigint NOT NULL CHECK (quantity_fails >= 0),
    description varchar(256) NOT NULL,
    price numeric(20, 6) CHECK (price > 0),
    dollar_proxy numeric(30, 6) GENERATED ALWAYS AS (quantity_fails * price) STORED,
    source_file_id bigint NOT NULL REFERENCES audit.source_file(source_file_id),
    source_row bigint NOT NULL,
    loaded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (settlement_date, cusip),
    UNIQUE (settlement_date, source_file_id, source_row)
) PARTITION BY RANGE (settlement_date);

CREATE TABLE IF NOT EXISTS core.ftd_observation_2026
    PARTITION OF core.ftd_observation
    FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');

CREATE TABLE IF NOT EXISTS core.ftd_observation_default
    PARTITION OF core.ftd_observation DEFAULT;

CREATE INDEX IF NOT EXISTS idx_ftd_cusip_date
    ON core.ftd_observation (cusip, settlement_date DESC);
CREATE INDEX IF NOT EXISTS idx_ftd_symbol_date
    ON core.ftd_observation (symbol, settlement_date DESC);
CREATE INDEX IF NOT EXISTS idx_ftd_date_brin
    ON core.ftd_observation USING brin (settlement_date) WITH (pages_per_range = 32);
CREATE INDEX IF NOT EXISTS idx_ftd_quantity_large
    ON core.ftd_observation (quantity_fails DESC)
    WHERE quantity_fails >= 100000;

CREATE TABLE IF NOT EXISTS quality.load_violation (
    violation_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    source_file_id bigint REFERENCES audit.source_file(source_file_id),
    source_row bigint,
    rule_code text NOT NULL,
    severity text NOT NULL CHECK (severity IN ('warning', 'error')),
    details jsonb NOT NULL,
    detected_at timestamptz NOT NULL DEFAULT clock_timestamp()
);

COMMENT ON COLUMN core.ftd_observation.quantity_fails IS
'Aggregate net fails-to-deliver balance reported for this security and settlement date; not new daily fails.';
COMMENT ON COLUMN core.ftd_observation.dollar_proxy IS
'Quantity multiplied by the SEC reference price. Descriptive proxy only; not a loss, liability, or transaction value.';

COMMIT;
