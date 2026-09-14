-- Cumulative concentration curve across securities for the selected corpus.
WITH ranked AS (
    SELECT
        cusip,
        representative_symbol AS symbol,
        reported_balance_observations,
        row_number() OVER (ORDER BY reported_balance_observations DESC, cusip) AS rank,
        sum(reported_balance_observations) OVER () AS market_total,
        sum(reported_balance_observations) OVER (
            ORDER BY reported_balance_observations DESC, cusip
            ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
        ) AS cumulative_quantity
    FROM mart.security_period_summary
)
SELECT
    rank,
    symbol,
    cusip,
    reported_balance_observations,
    round(cumulative_quantity::numeric / nullif(market_total, 0), 6) AS cumulative_share
FROM ranked
WHERE rank <= 100
ORDER BY rank;
