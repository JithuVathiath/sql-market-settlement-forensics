package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunHelpVersionAndUnknown(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"--help"}, {"version"}} {
		if err := run(args); err != nil {
			t.Fatalf("run(%v): %v", args, err)
		}
	}
	if err := run([]string{"unknown"}); err == nil {
		t.Fatal("expected unknown-command error")
	}
}

func TestRunNormalizeAndProfile(t *testing.T) {
	dir := t.TempDir()
	raw := filepath.Join(dir, "source.txt")
	canonical := filepath.Join(dir, "canonical.csv")
	rejects := filepath.Join(dir, "rejects.csv")
	receipt := filepath.Join(dir, "receipt.json")
	profileJSON := filepath.Join(dir, "profile.json")
	profileMarkdown := filepath.Join(dir, "profile.md")
	body := "SETTLEMENT DATE|CUSIP|SYMBOL|QUANTITY (FAILS)|DESCRIPTION|PRICE\n" +
		"20260701|037833100|AAPL|1200|APPLE INC|201.25\n"
	if err := os.WriteFile(raw, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"normalize", "--input", raw, "--output", canonical, "--rejects", rejects, "--receipt", receipt}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"profile", "--input", canonical, "--json", profileJSON, "--markdown", profileMarkdown}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(profileMarkdown)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "1,200 shares") {
		t.Fatalf("unexpected profile: %s", data)
	}
}

func TestRunRejectsBadFlags(t *testing.T) {
	if err := run([]string{"normalize", "--not-a-flag"}); err == nil {
		t.Fatal("expected flag error")
	}
	if err := run([]string{"profile", "--not-a-flag"}); err == nil {
		t.Fatal("expected flag error")
	}
	if err := run([]string{"fetch", "--not-a-flag"}); err == nil {
		t.Fatal("expected flag error")
	}
}
