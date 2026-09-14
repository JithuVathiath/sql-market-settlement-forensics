\set ON_ERROR_STOP on
\if :{?scale}
\else
\set scale 1000000
\endif

DROP TABLE IF EXISTS benchmark.ftd_heap;
CREATE UNLOGGED TABLE benchmark.ftd_heap (
    settlement_date date NOT NULL,
    cusip varchar(9) NOT NULL,
    symbol varchar(16) NOT NULL,
    quantity_fails bigint NOT NULL,
    price numeric(12, 4)
);

INSERT INTO benchmark.ftd_heap
SELECT
    date '2018-01-01' + ((n - 1) % 2200)::integer,
    lpad(((n * 7919) % 25000)::text, 9, '0'),
    'S' || ((n * 3571) % 25000)::text,
    100 + ((n * 104729) % 5000000),
    CASE WHEN n % 97 = 0 THEN NULL ELSE (1 + ((n * 8191) % 100000) / 100.0)::numeric(12, 4) END
FROM generate_series(1, :scale) AS series(n);

ANALYZE benchmark.ftd_heap;
