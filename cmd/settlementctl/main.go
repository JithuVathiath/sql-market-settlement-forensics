package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/JithuVathiath/sql-market-settlement-forensics/internal/secftd"
)

const version = "0.1.0"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}
	switch args[0] {
	case "version":
		fmt.Println(version)
		return nil
	case "normalize":
		set := flag.NewFlagSet("normalize", flag.ContinueOnError)
		input := set.String("input", "data/raw", "input file or directory")
		output := set.String("output", "data/processed/ftd.csv", "canonical CSV output")
		rejects := set.String("rejects", "data/rejects/ftd-rejects.csv", "quarantined rows")
		receipt := set.String("receipt", "data/processed/normalization-receipt.json", "normalization audit receipt")
		if err := set.Parse(args[1:]); err != nil {
			return err
		}
		stats, err := secftd.Normalize(*input, *output, *rejects)
		if err != nil {
			return err
		}
		if err := secftd.WriteNormalizeReceipt(stats, *receipt); err != nil {
			return err
		}
		fmt.Printf("files=%d accepted=%d rejected=%d\n", stats.Files, stats.Accepted, stats.Rejected)
		return nil
	case "profile":
		set := flag.NewFlagSet("profile", flag.ContinueOnError)
		input := set.String("input", "data/processed/ftd.csv", "canonical CSV input")
		jsonOut := set.String("json", "reports/generated/source-profile.json", "JSON profile output")
		markdownOut := set.String("markdown", "reports/generated/source-profile.md", "Markdown report output")
		if err := set.Parse(args[1:]); err != nil {
			return err
		}
		profile, err := secftd.BuildProfile(*input)
		if err != nil {
			return err
		}
		if err := secftd.WriteProfile(profile, *jsonOut, *markdownOut); err != nil {
			return err
		}
		fmt.Printf("rows=%d dates=%d cusips=%d\n", profile.Rows, profile.SettlementDates, profile.UniqueCUSIPs)
		return nil
	case "fetch":
		set := flag.NewFlagSet("fetch", flag.ContinueOnError)
		manifest := set.String("manifest", "data/source-manifest.json", "source manifest")
		output := set.String("output", "data/raw", "download directory")
		userAgent := set.String("user-agent", "", "descriptive SEC fair-access user agent")
		if err := set.Parse(args[1:]); err != nil {
			return err
		}
		client := &http.Client{Timeout: 2 * time.Minute}
		receipts, err := secftd.Fetch(context.Background(), client, *manifest, *output, *userAgent, time.Now())
		if err != nil {
			return err
		}
		fmt.Printf("downloaded=%d\n", len(receipts))
		return nil
	case "help", "--help", "-h":
		usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() {
	fmt.Print(`settlementctl — reproducible SEC settlement-data ingestion

Usage:
  settlementctl fetch     --user-agent "project-name contact-url"
  settlementctl normalize [--input path] [--output path] [--rejects path]
  settlementctl profile   [--input path] [--json path] [--markdown path]
  settlementctl version
`)
}
