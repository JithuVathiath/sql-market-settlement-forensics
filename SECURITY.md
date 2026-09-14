# Security Policy

Please report a suspected vulnerability privately through GitHub's security-advisory interface rather than opening a public issue.

Supported release: `v0.1.x`.

## Design Controls

- Downloader URLs must use HTTPS and resolve to the explicit SEC hostname allowlist.
- Downloads are capped at 500 MiB and written to a partial file before an atomic rename.
- Optional SHA-256 values fail closed on mismatch; every successful retrieval produces a receipt.
- ZIP member names are validated against path traversal and oversized members.
- Source fields are parsed as data and loaded through PostgreSQL `COPY`; they are never executed as SQL.
- Containers run the ingestion binary as a non-root user.

Do not commit raw archives, credentials, `.env` files, or database volumes.
