-- How much of each day's reported balance sits in the ten largest observations?
SELECT
    settlement_date,
    daily_total AS reported_balance_observations,
    top_ten_quantity,
    top_ten_share,
    percent_rank() OVER (ORDER BY top_ten_share) AS concentration_percentile
FROM mart.daily_concentration
ORDER BY top_ten_share DESC, settlement_date;

