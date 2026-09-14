-- Day-over-day changes in reported aggregate balances for recurring CUSIPs.
WITH changes AS (
    SELECT
        settlement_date,
        cusip,
        symbol,
        quantity_fails,
        lag(quantity_fails) OVER (PARTITION BY cusip ORDER BY settlement_date) AS prior_quantity,
        settlement_date - lag(settlement_date) OVER (
            PARTITION BY cusip ORDER BY settlement_date
        ) AS calendar_gap
    FROM core.ftd_observation
)
SELECT
    settlement_date,
    symbol,
    cusip,
    prior_quantity,
    quantity_fails,
    quantity_fails - prior_quantity AS absolute_change,
    round((quantity_fails - prior_quantity)::numeric / nullif(prior_quantity, 0), 4) AS relative_change,
    calendar_gap
FROM changes
WHERE prior_quantity IS NOT NULL
ORDER BY abs(quantity_fails - prior_quantity) DESC
LIMIT 100;

