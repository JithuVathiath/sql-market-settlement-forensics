# Methodology

## Unit Of Analysis

One core row is one SEC-reported aggregate net balance for a CUSIP on a settlement date. The natural key is `(settlement_date, cusip)`. Ticker and description are treated as time-varying labels, not stable identifiers.

## Pipeline

1. **Acquire:** read a reviewed source manifest, enforce HTTPS and the SEC host allowlist, cap downloads at 500 MiB, compute SHA-256, and write response receipts.
2. **Normalize:** validate the six-column header, stream rows without loading an archive into memory, convert `YYYYMMDD` dates, uppercase identifiers, map `.` prices to missing, and quarantine malformed/control rows.
3. **Stage:** preserve source filename and row number in an untyped raw layer.
4. **Type:** cast into constrained, yearly partitioned core storage and generate `quantity × price` only when price exists.
5. **Model:** refresh auditable marts for daily distributions, security summaries, observation streaks, concentration, identity changes, and review flags.
6. **Test:** execute domain, uniqueness, lineage, non-empty, and query-compatibility assertions.

## Measures

- **Reported balance observations:** the sum of daily aggregate net balances. The name makes the non-additive interpretation explicit.
- **Dollar proxy:** quantity multiplied by the SEC reference price. Missing price yields missing proxy, not zero at row level.
- **Observation streak:** consecutive available settlement dates on which a CUSIP appears. It does not establish that the same underlying fail remained open.
- **Historical review flag:** a quantity at least three sample standard deviations above up to twenty preceding observations, with at least five prior observations and a 100,000-share floor.
- **Top-ten concentration:** share of a date's reported balance attributable to the ten largest CUSIP observations.

## Reproducibility

The small fixture keeps CI deterministic and tests edge cases without downloading live data. The committed real-corpus report can be regenerated from the immutable validation manifest. Production downloads are not run in CI because repeatedly fetching unchanged public archives would make builds depend on an external service and conflict with good API citizenship.
