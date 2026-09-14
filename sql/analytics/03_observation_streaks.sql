-- Repeated appearance is called an observation streak, not persistence of one fail.
SELECT
    symbol,
    cusip,
    streak_start,
    streak_end,
    consecutive_observation_dates,
    reported_balance_observations,
    peak_reported_balance
FROM mart.observation_streaks
WHERE consecutive_observation_dates >= 3
ORDER BY consecutive_observation_dates DESC, peak_reported_balance DESC
LIMIT 50;

