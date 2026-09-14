\set ON_ERROR_STOP on

BEGIN;
TRUNCATE raw.import_buffer;

\copy raw.import_buffer FROM 'data/processed/ftd.csv' WITH (FORMAT csv, HEADER true)

INSERT INTO audit.source_file (source_name, raw_row_count, accepted_row_count, provenance)
SELECT
    source_file,
    count(*),
    count(*),
    jsonb_build_object('loader', 'sql/load_canonical.sql', 'input', 'data/processed/ftd.csv')
FROM raw.import_buffer
GROUP BY source_file
ON CONFLICT (source_name) DO UPDATE SET
    raw_row_count = EXCLUDED.raw_row_count,
    accepted_row_count = EXCLUDED.accepted_row_count,
    provenance = audit.source_file.provenance || EXCLUDED.provenance;

INSERT INTO raw.ftd_observation (
    source_file_id, source_row, settlement_date_text, cusip_text, symbol_text,
    quantity_fails_text, description_text, price_text
)
SELECT
    f.source_file_id,
    b.source_row::bigint,
    b.settlement_date,
    b.cusip,
    b.symbol,
    b.quantity_fails,
    b.description,
    nullif(b.price, '')
FROM raw.import_buffer b
JOIN audit.source_file f ON f.source_name = b.source_file
ON CONFLICT (source_file_id, source_row) DO UPDATE SET
    settlement_date_text = EXCLUDED.settlement_date_text,
    cusip_text = EXCLUDED.cusip_text,
    symbol_text = EXCLUDED.symbol_text,
    quantity_fails_text = EXCLUDED.quantity_fails_text,
    description_text = EXCLUDED.description_text,
    price_text = EXCLUDED.price_text;

INSERT INTO core.ftd_observation (
    settlement_date, cusip, symbol, quantity_fails, description, price,
    source_file_id, source_row
)
SELECT
    r.settlement_date_text::date,
    upper(r.cusip_text),
    upper(r.symbol_text),
    r.quantity_fails_text::bigint,
    r.description_text,
    nullif(r.price_text, '')::numeric,
    r.source_file_id,
    r.source_row
FROM raw.ftd_observation r
ON CONFLICT (settlement_date, cusip) DO UPDATE SET
    symbol = EXCLUDED.symbol,
    quantity_fails = EXCLUDED.quantity_fails,
    description = EXCLUDED.description,
    price = EXCLUDED.price,
    source_file_id = EXCLUDED.source_file_id,
    source_row = EXCLUDED.source_row,
    loaded_at = clock_timestamp();

REFRESH MATERIALIZED VIEW mart.security_month_summary;
COMMIT;
