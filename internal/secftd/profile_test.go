package secftd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAndWriteProfile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "canonical.csv")
	body := "settlement_date,cusip,symbol,quantity_fails,description,price,source_file,source_row\n" +
		"2026-07-01,000000001,AAA,100,ALPHA,2.50,a.txt,2\n" +
		"2026-07-02,000000001,AAA,300,ALPHA,3.00,a.txt,3\n" +
		"2026-07-02,000000002,BBB,50,BETA,,a.txt,4\n"
	if err := os.WriteFile(input, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	profile, err := BuildProfile(input)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Rows != 3 || profile.UniqueCUSIPs != 2 || profile.SettlementDates != 2 || profile.MissingPriceRows != 1 || profile.QuantityFails != 450 {
		t.Fatalf("unexpected profile: %#v", profile)
	}
	if profile.DollarProxy != 1150 {
		t.Fatalf("unexpected proxy: %v", profile.DollarProxy)
	}
	jsonPath := filepath.Join(dir, "nested", "profile.json")
	mdPath := filepath.Join(dir, "nested", "profile.md")
	if err := WriteProfile(profile, jsonPath, mdPath); err != nil {
		t.Fatal(err)
	}
	markdown, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(markdown), "not evidence") || !strings.Contains(string(markdown), "$1,150.00") || !strings.Contains(string(markdown), "Unavailable") {
		t.Fatalf("missing interpretation or metric: %s", markdown)
	}
}

func TestBuildProfileRejectsEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.csv")
	if err := os.WriteFile(path, []byte("settlement_date,cusip,symbol,quantity_fails,description,price,source_file,source_row\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildProfile(path); err == nil {
		t.Fatal("expected empty-profile error")
	}
}
