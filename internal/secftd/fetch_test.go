package secftd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func TestFetchWritesVerifiedArtifactAndReceipt(t *testing.T) {
	payload := []byte("PK synthetic zip bytes")
	sum := sha256.Sum256(payload)
	digest := hex.EncodeToString(sum[:])
	dir := t.TempDir()
	manifest := filepath.Join(dir, "manifest.json")
	manifestBody := `{"dataset":"SEC FTD","files":[{"period":"2026-07","half":"first","url":"https://www.sec.gov/files/a.zip","sha256":"` + digest + `"}]}`
	if err := os.WriteFile(manifest, []byte(manifestBody), 0o600); err != nil {
		t.Fatal(err)
	}
	var seenAgent string
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		seenAgent = req.Header.Get("User-Agent")
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(payload)), Header: http.Header{"Etag": []string{"test-etag"}}, ContentLength: int64(len(payload))}, nil
	})}
	now := time.Date(2026, 9, 14, 9, 0, 0, 0, time.UTC)
	receipts, err := Fetch(context.Background(), client, manifest, filepath.Join(dir, "raw"), "test-project example.invalid", now)
	if err != nil {
		t.Fatal(err)
	}
	if seenAgent == "" || len(receipts) != 1 || receipts[0].SHA256 != digest {
		t.Fatalf("unexpected receipt: %#v", receipts)
	}
	got, err := os.ReadFile(filepath.Join(dir, "raw", "a.zip"))
	if err != nil || !bytes.Equal(got, payload) {
		t.Fatalf("artifact mismatch: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "raw", "download-receipts.json")); err != nil {
		t.Fatal(err)
	}
}

func TestFetchSafetyChecks(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"dataset":"x","files":[{"period":"x","half":"first","url":"https://example.com/file.zip"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Fetch(context.Background(), &http.Client{}, manifest, filepath.Join(dir, "raw"), "", time.Now()); err == nil || !strings.Contains(err.Error(), "user-agent") {
		t.Fatalf("expected user-agent error, got %v", err)
	}
	if _, err := Fetch(context.Background(), &http.Client{}, manifest, filepath.Join(dir, "raw"), "test", time.Now()); err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("expected allowlist error, got %v", err)
	}
}

func TestFetchRejectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(manifest, []byte(`{"dataset":"x","files":[{"period":"x","half":"first","url":"https://www.sec.gov/file.zip","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Status: "200 OK", Body: io.NopCloser(strings.NewReader("bad")), Header: make(http.Header), ContentLength: 3}, nil
	})}
	if _, err := Fetch(context.Background(), client, manifest, filepath.Join(dir, "raw"), "test", time.Now()); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum error, got %v", err)
	}
}
