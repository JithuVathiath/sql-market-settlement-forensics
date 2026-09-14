package secftd

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePlainFile(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.txt")
	data := strings.Join(expectedHeader, "|") + "\n" +
		"20260701|037833100|aapl|1200|APPLE INC|201.25\n" +
		"bad|037833100|AAPL|1200|APPLE INC|201.25\n"
	if err := os.WriteFile(input, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(dir, "nested", "output.csv")
	rejects := filepath.Join(dir, "nested", "rejects.csv")
	stats, err := Normalize(input, output, rejects)
	if err != nil {
		t.Fatal(err)
	}
	if stats.Files != 1 || stats.Accepted != 1 || stats.Rejected != 1 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
	if len(stats.Sources) != 1 || stats.Sources[0].Accepted != 1 || stats.Sources[0].Rejected != 1 {
		t.Fatalf("unexpected source stats: %#v", stats.Sources)
	}
	if err := WriteNormalizeReceipt(stats, filepath.Join(dir, "receipt", "normalization.json")); err != nil {
		t.Fatal(err)
	}
	rows := readCSV(t, output)
	if len(rows) != 2 || rows[1][2] != "AAPL" || rows[1][5] != "201.25" {
		t.Fatalf("unexpected output: %#v", rows)
	}
	rejected := readCSV(t, rejects)
	if len(rejected) != 2 || rejected[1][2] != "invalid settlement date" {
		t.Fatalf("unexpected rejects: %#v", rejected)
	}
}

func TestNormalizeZipWithMultipleMembers(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "two.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	for _, name := range []string{"a.txt", "nested/b.txt"} {
		member, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = member.Write([]byte(strings.Join(expectedHeader, "|") + "\n20260701|037833100|AAPL|1|APPLE INC|1.00\n"))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	stats, err := Normalize(archivePath, filepath.Join(dir, "out.csv"), filepath.Join(dir, "reject.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if stats.Accepted != 2 || stats.Files != 2 {
		t.Fatalf("unexpected stats: %#v", stats)
	}
}

func TestDiscoverInputsRejectsUnsafeArchiveMember(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "unsafe.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(file)
	member, _ := zw.Create("../escape.txt")
	_, _ = member.Write([]byte("anything"))
	_ = zw.Close()
	_ = file.Close()
	if _, err := discoverInputs(path); err == nil || !strings.Contains(err.Error(), "unsafe archive member") {
		t.Fatalf("expected unsafe member rejection, got %v", err)
	}
}

func TestNormalizeNoInputs(t *testing.T) {
	dir := t.TempDir()
	if _, err := Normalize(dir, filepath.Join(dir, "out.csv"), filepath.Join(dir, "reject.csv")); err == nil {
		t.Fatal("expected no-input error")
	}
}

func readCSV(t *testing.T, path string) [][]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	return rows
}
