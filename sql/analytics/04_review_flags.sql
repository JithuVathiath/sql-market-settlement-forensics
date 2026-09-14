-- Statistical review queue. A flag is descriptive and never an allegation.
SELECT
    settlement_date,
    symbol,
    cusip,
    quantity_fails,
    prior_mean,
    historical_z_score,
    review_flag
FROM mart.quantity_review_flags
ORDER BY historical_z_score DESC, quantity_fails DESC;
