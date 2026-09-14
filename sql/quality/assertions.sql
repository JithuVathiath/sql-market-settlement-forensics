\set ON_ERROR_STOP on

DO $$
DECLARE
    violation_count bigint;
BEGIN
    SELECT count(*) INTO violation_count
    FROM core.ftd_observation
    WHERE quantity_fails < 0
       OR length(cusip) NOT BETWEEN 1 AND 9
       OR symbol = ''
       OR (price IS NOT NULL AND price <= 0)
       OR source_row <= 1;
    IF violation_count > 0 THEN
        RAISE EXCEPTION 'core domain assertions failed for % rows', violation_count;
    END IF;

    SELECT count(*) INTO violation_count
    FROM (
        SELECT settlement_date_text, cusip_text
        FROM raw.ftd_observation
        GROUP BY settlement_date_text, cusip_text
        HAVING count(*) > 1
    ) duplicates;
    IF violation_count > 0 THEN
        RAISE EXCEPTION 'raw natural-key duplicates detected for % keys', violation_count;
    END IF;

    SELECT count(*) INTO violation_count
    FROM raw.ftd_observation r
    LEFT JOIN core.ftd_observation c
      ON c.source_file_id = r.source_file_id AND c.source_row = r.source_row
    WHERE c.source_file_id IS NULL;
    IF violation_count > 0 THEN
        RAISE EXCEPTION 'lineage assertion failed for % raw rows', violation_count;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM core.ftd_observation) THEN
        RAISE EXCEPTION 'warehouse is empty';
    END IF;
END $$;

SELECT 'quality assertions passed' AS result;
