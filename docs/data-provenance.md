# Data Provenance

## Production Source

The authority is the U.S. Securities and Exchange Commission's [Fails-to-Deliver Data](https://www.sec.gov/data-research/sec-markets-data/fails-deliver-data) page. The production manifest points directly to official SEC HTTPS archives.

Starting in July 2009, the SEC publishes a first-half and second-half file for each month. Each pipe-delimited row contains settlement date, CUSIP, ticker, quantity, description, and a reference price. A dot in the price field is normalized to SQL `NULL`, never zero.

## February 2021 Validation Corpus

The committed report uses complete February 2021 total-market files preserved in the public `tangentstorm/gme-data` repository. The mirror's own documentation says its `raw/sec/ftd` directory contains raw total-market SEC files and its extraction script reads those files without rewriting them.

The validation corpus is pinned to commit `0e151cfdab3425d4bf2d05fe337c8f8eac494347`. [`data/validation-manifest.json`](../data/validation-manifest.json) records:

- the immutable raw URLs;
- Git blob identifiers;
- independently calculated SHA-256 values;
- each file's business-row and quantity control totals.

This mirror is a reproducibility fallback, not a replacement authority. A future refresh should use `data/source-manifest.json` against the SEC directly.

## Reconciliation

Each February file ends with two control rows. The normalizer quarantined those non-business rows and produced these exact checks:

| File | Accepted records | Trailer quantity | SHA-256 prefix |
|---|---:|---:|---|
| `cnsfails202102a.txt` | 59,224 | 3,242,854,991 | `610d305a0ff3…` |
| `cnsfails202102b.txt` | 54,843 | 4,088,130,057 | `ff53ead16768…` |
| **Total** | **114,067** | **7,330,985,048** | — |

The accepted-record and aggregate-quantity results exactly match both trailer controls. Raw archives and normalized row-level data are intentionally excluded from Git; compact manifests and computed results are committed.

## Historical Comparability

Before 16 September 2008, the SEC excluded balances below 10,000 shares. Analyses crossing that date require an explicit regime-break indicator. The committed February 2021 validation period is unaffected by that historical threshold.
