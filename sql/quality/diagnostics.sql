-- Data-quality diagnostics return zero rows when the loaded corpus is clean.
SELECT 'duplicate_natural_key' AS rule_code, settlement_date, cusip, count(*) AS occurrences
FROM core.ftd_observation
GROUP BY settlement_date, cusip
HAVING count(*) > 1;

SELECT 'missing_price' AS rule_code, settlement_date, cusip, symbol
FROM core.ftd_observation
WHERE price IS NULL
ORDER BY settlement_date, cusip;

SELECT
    source_name,
    raw_row_count,
    accepted_row_count,
    rejected_row_count,
    raw_row_count - accepted_row_count - rejected_row_count AS unexplained_rows
FROM audit.source_file
WHERE raw_row_count <> accepted_row_count + rejected_row_count;

