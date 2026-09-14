BEGIN;

CREATE OR REPLACE VIEW mart.daily_market_summary AS
SELECT
    settlement_date,
    count(*) AS security_observations,
    count(*) FILTER (WHERE price IS NULL) AS missing_price_observations,
    count(DISTINCT cusip) AS distinct_cusips,
    sum(quantity_fails) AS reported_balance_observations,
    sum(dollar_proxy) AS dollar_proxy,
    percentile_cont(0.5) WITHIN GROUP (ORDER BY quantity_fails) AS median_quantity,
    percentile_cont(0.95) WITHIN GROUP (ORDER BY quantity_fails) AS p95_quantity
FROM core.ftd_observation
GROUP BY settlement_date;

CREATE OR REPLACE VIEW mart.security_period_summary AS
SELECT
    cusip,
    mode() WITHIN GROUP (ORDER BY symbol) AS representative_symbol,
    max(description) AS representative_description,
    min(settlement_date) AS first_observed_date,
    max(settlement_date) AS last_observed_date,
    count(*) AS observation_days,
    sum(quantity_fails) AS reported_balance_observations,
    max(quantity_fails) AS peak_reported_balance,
    avg(quantity_fails)::numeric(30, 2) AS mean_reported_balance,
    sum(dollar_proxy) AS dollar_proxy,
    count(*) FILTER (WHERE price IS NULL) AS missing_price_observations
FROM core.ftd_observation
GROUP BY cusip;

CREATE OR REPLACE VIEW mart.observation_streaks AS
WITH market_dates AS (
    SELECT settlement_date,
           dense_rank() OVER (ORDER BY settlement_date) AS market_day_number
    FROM (SELECT DISTINCT settlement_date FROM core.ftd_observation) dates
), sequenced AS (
    SELECT
        f.cusip,
        f.symbol,
        f.settlement_date,
        f.quantity_fails,
        d.market_day_number,
        d.market_day_number - row_number() OVER (
            PARTITION BY f.cusip ORDER BY f.settlement_date
        ) AS island
    FROM core.ftd_observation f
    JOIN market_dates d USING (settlement_date)
), streaks AS (
    SELECT
        cusip,
        mode() WITHIN GROUP (ORDER BY symbol) AS symbol,
        min(settlement_date) AS streak_start,
        max(settlement_date) AS streak_end,
        count(*) AS consecutive_observation_dates,
        sum(quantity_fails) AS reported_balance_observations,
        max(quantity_fails) AS peak_reported_balance
    FROM sequenced
    GROUP BY cusip, island
)
SELECT * FROM streaks;

COMMENT ON VIEW mart.observation_streaks IS
'Consecutive dates on which a CUSIP appears. A streak does not establish that the same underlying fail persisted.';

CREATE OR REPLACE VIEW mart.quantity_review_flags AS
WITH history AS (
    SELECT
        f.*,
        count(*) OVER prior AS prior_observations,
        avg(quantity_fails) OVER prior AS prior_mean,
        stddev_samp(quantity_fails) OVER prior AS prior_stddev
    FROM core.ftd_observation f
    WINDOW prior AS (
        PARTITION BY cusip ORDER BY settlement_date
        ROWS BETWEEN 20 PRECEDING AND 1 PRECEDING
    )
), scored AS (
    SELECT
        *,
        CASE
            WHEN prior_observations >= 5 AND prior_stddev > 0
            THEN (quantity_fails - prior_mean) / prior_stddev
        END AS historical_z_score
    FROM history
)
SELECT
    settlement_date,
    cusip,
    symbol,
    quantity_fails,
    round(prior_mean, 2) AS prior_mean,
    round(historical_z_score, 3) AS historical_z_score,
    'descriptive_quantity_spike'::text AS review_flag
FROM scored
WHERE historical_z_score >= 3
  AND quantity_fails >= 100000;

COMMENT ON VIEW mart.quantity_review_flags IS
'Descriptive statistical review flags. They are not allegations or evidence of manipulation or misconduct.';

CREATE OR REPLACE VIEW mart.daily_concentration AS
WITH ranked AS (
    SELECT
        settlement_date,
        cusip,
        symbol,
        quantity_fails,
        row_number() OVER (
            PARTITION BY settlement_date ORDER BY quantity_fails DESC, cusip
        ) AS quantity_rank,
        sum(quantity_fails) OVER (PARTITION BY settlement_date) AS daily_total
    FROM core.ftd_observation
)
SELECT
    settlement_date,
    daily_total,
    sum(quantity_fails) FILTER (WHERE quantity_rank <= 10) AS top_ten_quantity,
    round(
        sum(quantity_fails) FILTER (WHERE quantity_rank <= 10)::numeric
        / nullif(daily_total, 0),
        6
    ) AS top_ten_share
FROM ranked
GROUP BY settlement_date, daily_total;

CREATE OR REPLACE VIEW mart.identity_change_log AS
WITH changes AS (
    SELECT
        settlement_date,
        cusip,
        symbol,
        description,
        lag(symbol) OVER (PARTITION BY cusip ORDER BY settlement_date) AS prior_symbol,
        lag(description) OVER (PARTITION BY cusip ORDER BY settlement_date) AS prior_description
    FROM core.ftd_observation
)
SELECT *
FROM changes
WHERE (prior_symbol IS NOT NULL AND symbol IS DISTINCT FROM prior_symbol)
   OR (prior_description IS NOT NULL AND description IS DISTINCT FROM prior_description);

CREATE MATERIALIZED VIEW IF NOT EXISTS mart.security_month_summary AS
SELECT
    date_trunc('month', settlement_date)::date AS settlement_month,
    cusip,
    mode() WITHIN GROUP (ORDER BY symbol) AS representative_symbol,
    count(*) AS observation_days,
    sum(quantity_fails) AS reported_balance_observations,
    max(quantity_fails) AS peak_reported_balance,
    sum(dollar_proxy) AS dollar_proxy
FROM core.ftd_observation
GROUP BY date_trunc('month', settlement_date)::date, cusip
WITH NO DATA;

CREATE UNIQUE INDEX IF NOT EXISTS idx_security_month_summary_pk
    ON mart.security_month_summary (settlement_month, cusip);

COMMIT;
