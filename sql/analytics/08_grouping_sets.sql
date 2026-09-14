-- One scan produces daily, symbol, and grand-total summaries.
SELECT
    CASE
        WHEN grouping(settlement_date) = 0 THEN 'date'
        WHEN grouping(symbol) = 0 THEN 'symbol'
        ELSE 'all'
    END AS grain,
    settlement_date,
    symbol,
    count(*) AS observations,
    sum(quantity_fails) AS reported_balance_observations,
    round(sum(dollar_proxy), 2) AS dollar_proxy
FROM core.ftd_observation
GROUP BY GROUPING SETS ((settlement_date), (symbol), ())
ORDER BY grain, reported_balance_observations DESC;

