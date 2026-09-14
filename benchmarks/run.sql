\set ON_ERROR_STOP on
\timing on

\echo 'Baseline: no secondary index'
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT settlement_date, quantity_fails
FROM benchmark.ftd_heap
WHERE cusip = '000007919'
ORDER BY settlement_date DESC;

\echo 'B-tree: point history by CUSIP and date'
CREATE INDEX IF NOT EXISTS benchmark_ftd_cusip_date
    ON benchmark.ftd_heap (cusip, settlement_date DESC)
    INCLUDE (quantity_fails);
ANALYZE benchmark.ftd_heap;
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT settlement_date, quantity_fails
FROM benchmark.ftd_heap
WHERE cusip = '000007919'
ORDER BY settlement_date DESC;

\echo 'BRIN: compact support for append-oriented date ranges'
CREATE INDEX IF NOT EXISTS benchmark_ftd_date_brin
    ON benchmark.ftd_heap USING brin (settlement_date) WITH (pages_per_range = 32);
ANALYZE benchmark.ftd_heap;
EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)
SELECT count(*), sum(quantity_fails)
FROM benchmark.ftd_heap
WHERE settlement_date BETWEEN date '2023-01-01' AND date '2023-01-31';

