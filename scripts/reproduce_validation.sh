#!/bin/sh
set -eu

root_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
raw_dir="$root_dir/data/raw/february-2021-validation"
processed="$root_dir/data/processed/february-2021.csv"
rejects="$root_dir/data/rejects/february-2021.csv"
generated="$root_dir/reports/generated"

mkdir -p "$raw_dir" "$generated"

curl --fail --location --retry 3 \
  --user-agent "sql-market-settlement-forensics/0.1 github.com/JithuVathiath/sql-market-settlement-forensics" \
  --output "$raw_dir/cnsfails202102a.txt" \
  "https://raw.githubusercontent.com/tangentstorm/gme-data/0e151cfdab3425d4bf2d05fe337c8f8eac494347/raw/sec/ftd/cnsfails202102a.txt"

curl --fail --location --retry 3 \
  --user-agent "sql-market-settlement-forensics/0.1 github.com/JithuVathiath/sql-market-settlement-forensics" \
  --output "$raw_dir/cnsfails202102b.txt" \
  "https://raw.githubusercontent.com/tangentstorm/gme-data/0e151cfdab3425d4bf2d05fe337c8f8eac494347/raw/sec/ftd/cnsfails202102b.txt"

if command -v sha256sum >/dev/null 2>&1; then
  echo "610d305a0ff3c0d136688c3df531742705a93c68231ab09b338652dbdd62bece  $raw_dir/cnsfails202102a.txt" | sha256sum --check
  echo "ff53ead16768451a40c9109d38961a3bbedc50866814eba6823777c2e0f620d8  $raw_dir/cnsfails202102b.txt" | sha256sum --check
else
  test "$(shasum -a 256 "$raw_dir/cnsfails202102a.txt" | awk '{print $1}')" = "610d305a0ff3c0d136688c3df531742705a93c68231ab09b338652dbdd62bece"
  test "$(shasum -a 256 "$raw_dir/cnsfails202102b.txt" | awk '{print $1}')" = "ff53ead16768451a40c9109d38961a3bbedc50866814eba6823777c2e0f620d8"
fi

cd "$root_dir"
go run ./cmd/settlementctl normalize \
  --input "$raw_dir" \
  --output "$processed" \
  --rejects "$rejects" \
  --receipt "$generated/february-2021-normalization.json"

go run ./cmd/settlementctl profile \
  --input "$processed" \
  --json "$generated/february-2021-profile.json" \
  --markdown "$generated/february-2021-profile.md"

cmp "$generated/february-2021-profile.json" "$root_dir/reports/february-2021-profile.json"
echo "Validation profile reproduced exactly."
