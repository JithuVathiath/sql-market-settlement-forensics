# Reproduce The Real-Corpus Report

The repository does not commit approximately seven megabytes of raw row-level source data. It commits immutable identifiers, hashes, control totals, and derived results instead.

Run:

```bash
./scripts/reproduce_validation.sh
```

The script downloads both complete February 2021 SEC-format text files from the pinned Git commit, verifies each SHA-256 digest, streams 114,067 business records through the normalizer, quarantines four trailer-control rows, regenerates the JSON and Markdown profiles, and byte-compares the generated JSON with the committed result.

This validation route exists because historical public archives are useful for deterministic replication. New production analyses should use the official SEC manifest and downloader described in the main README.
