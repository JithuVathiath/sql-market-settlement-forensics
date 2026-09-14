# Performance Experiment

The benchmark creates a deterministic, one-million-row PostgreSQL table with 25,000 synthetic securities across 2,200 dates. It compares two investigation access patterns:

1. security history ordered by date;
2. aggregate quantity across a bounded date range.

`benchmarks/run.sql` captures execution time, plan shape, planning time, buffer hits, and reads before and after adding:

- a covering B-tree on `(cusip, settlement_date DESC) INCLUDE (quantity_fails)`;
- a compact BRIN on `settlement_date` with 32 pages per range.

Run `make benchmark` to create `benchmarks/results/latest.txt`. CI also runs a 100,000-row smoke version and uploads its plan as a build artifact. Results are measured on the executing machine rather than advertised as universal numbers; cache warmth, storage, PostgreSQL version, and data scale all affect timing.
