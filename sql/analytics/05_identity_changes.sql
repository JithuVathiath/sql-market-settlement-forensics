-- Surface ticker/description changes so analysts do not join on ticker alone.
SELECT
    settlement_date,
    cusip,
    prior_symbol,
    symbol,
    prior_description,
    description
FROM mart.identity_change_log
ORDER BY settlement_date, cusip;
