package secftd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxDownloadBytes int64 = 500 << 20

type SourceManifest struct {
	Dataset   string       `json:"dataset"`
	Authority string       `json:"authority"`
	Landing   string       `json:"landing_page"`
	Files     []SourceFile `json:"files"`
}

type SourceFile struct {
	Period string `json:"period"`
	Half   string `json:"half"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256,omitempty"`
}

type Receipt struct {
	Period       string `json:"period"`
	Half         string `json:"half"`
	URL          string `json:"url"`
	File         string `json:"file"`
	SHA256       string `json:"sha256"`
	Bytes        int64  `json:"bytes"`
	LastModified string `json:"last_modified,omitempty"`
	ETag         string `json:"etag,omitempty"`
	RetrievedAt  string `json:"retrieved_at"`
}

func LoadManifest(path string) (SourceManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return SourceManifest{}, err
	}
	var manifest SourceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return SourceManifest{}, err
	}
	if manifest.Dataset == "" || len(manifest.Files) == 0 {
		return SourceManifest{}, fmt.Errorf("manifest requires a dataset and at least one file")
	}
	return manifest, nil
}

func Fetch(ctx context.Context, client *http.Client, manifestPath, outputDir, userAgent string, now time.Time) ([]Receipt, error) {
	if strings.TrimSpace(userAgent) == "" {
		return nil, fmt.Errorf("a descriptive --user-agent is required by SEC fair-access guidance")
	}
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, err
	}
	var receipts []Receipt
	for index, file := range manifest.Files {
		if index > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(120 * time.Millisecond):
			}
		}
		receipt, err := fetchOne(ctx, client, file, outputDir, userAgent, now)
		if err != nil {
			return nil, fmt.Errorf("fetch %s %s: %w", file.Period, file.Half, err)
		}
		receipts = append(receipts, receipt)
	}
	data, err := json.MarshalIndent(receipts, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(outputDir, "download-receipts.json"), data, 0o644); err != nil {
		return nil, err
	}
	return receipts, nil
}

func fetchOne(ctx context.Context, client *http.Client, source SourceFile, outputDir, userAgent string, now time.Time) (Receipt, error) {
	parsed, err := url.Parse(source.URL)
	if err != nil || parsed.Scheme != "https" {
		return Receipt{}, fmt.Errorf("source must be a valid HTTPS URL")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "sec.gov" && host != "www.sec.gov" {
		return Receipt{}, fmt.Errorf("source host %q is not on the SEC allowlist", host)
	}
	name := filepath.Base(parsed.Path)
	if name == "." || name == "/" || name == "" {
		return Receipt{}, fmt.Errorf("source URL has no filename")
	}
	destination := filepath.Join(outputDir, name)
	temporary := destination + ".partial"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		return Receipt{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/zip, application/octet-stream;q=0.9")
	response, err := client.Do(req)
	if err != nil {
		return Receipt{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Receipt{}, fmt.Errorf("unexpected HTTP status %s", response.Status)
	}
	if response.ContentLength > maxDownloadBytes {
		return Receipt{}, fmt.Errorf("response exceeds %d-byte safety limit", maxDownloadBytes)
	}

	out, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return Receipt{}, err
	}
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(out, hash), io.LimitReader(response.Body, maxDownloadBytes+1))
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(temporary)
		return Receipt{}, copyErr
	}
	if closeErr != nil {
		_ = os.Remove(temporary)
		return Receipt{}, closeErr
	}
	if written > maxDownloadBytes {
		_ = os.Remove(temporary)
		return Receipt{}, fmt.Errorf("response exceeded %d-byte safety limit", maxDownloadBytes)
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	if source.SHA256 != "" && !strings.EqualFold(source.SHA256, digest) {
		_ = os.Remove(temporary)
		return Receipt{}, fmt.Errorf("checksum mismatch: expected %s, got %s", source.SHA256, digest)
	}
	if err := os.Rename(temporary, destination); err != nil {
		_ = os.Remove(temporary)
		return Receipt{}, err
	}
	return Receipt{
		Period: source.Period, Half: source.Half, URL: source.URL, File: name,
		SHA256: digest, Bytes: written, LastModified: response.Header.Get("Last-Modified"),
		ETag: response.Header.Get("ETag"), RetrievedAt: now.UTC().Format(time.RFC3339),
	}, nil
}
