-- Daily settlement-stress pulse with robust distribution statistics.
SELECT
    settlement_date,
    security_observations,
    distinct_cusips,
    reported_balance_observations,
    round(dollar_proxy, 2) AS dollar_proxy,
    round(median_quantity::numeric, 2) AS median_quantity,
    round(p95_quantity::numeric, 2) AS p95_quantity,
    round(missing_price_observations::numeric / nullif(security_observations, 0), 4) AS missing_price_rate
FROM mart.daily_market_summary
ORDER BY settlement_date;

