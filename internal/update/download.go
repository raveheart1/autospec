package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Downloader handles downloading and verifying release binaries.
type Downloader struct {
	httpClient *http.Client
}

// NewDownloader creates a new downloader with the given HTTP client.
func NewDownloader(client *http.Client) *Downloader {
	if client == nil {
		client = &http.Client{Timeout: DefaultHTTPTimeout}
	}
	return &Downloader{httpClient: client}
}

// ProgressWriter wraps an io.Writer to report download progress.
type ProgressWriter struct {
	Writer   io.Writer
	Total    int64
	Current  int64
	OnUpdate func(current, total int64)
}

// Write implements io.Writer and reports progress.
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	pw.Current += int64(n)
	if pw.OnUpdate != nil {
		pw.OnUpdate(pw.Current, pw.Total)
	}
	return n, err
}

// DownloadBinary downloads a binary archive from the given URL to a temp file.
// Returns the path to the downloaded file.
func (d *Downloader) DownloadBinary(ctx context.Context, url string, onProgress func(current, total int64)) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "autospec-update-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer tmpFile.Close()

	writer := io.Writer(tmpFile)
	if onProgress != nil {
		writer = &ProgressWriter{
			Writer:   tmpFile,
			Total:    resp.ContentLength,
			OnUpdate: onProgress,
		}
	}

	if _, err := io.Copy(writer, resp.Body); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("writing to temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// FetchChecksum downloads the checksums.txt file and returns the checksum for the given asset.
func (d *Downloader) FetchChecksum(ctx context.Context, checksumURL, assetName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating checksum request: %w", err)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching checksums: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("checksum fetch failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading checksum body: %w", err)
	}

	return ParseChecksum(string(body), assetName)
}

// ParseChecksum extracts the checksum for a specific asset from checksums.txt format.
// Format: "<checksum>  <filename>"
func ParseChecksum(content, assetName string) (string, error) {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "sha256hash  filename" (two spaces between hash and filename)
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == assetName {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("checksum not found for asset: %s", assetName)
}

// VerifyChecksum computes the SHA256 hash of a file and compares it to expected.
func VerifyChecksum(filePath, expected string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("opening file for checksum: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("computing checksum: %w", err)
	}

	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
	}

	return nil
}

// ExtractBinary extracts the autospec binary from a tar.gz archive.
// Returns the path to the extracted binary.
func ExtractBinary(archivePath, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("opening archive: %w", err)
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("reading tar entry: %w", err)
		}

		// Look for the autospec binary (not a directory)
		if header.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(header.Name) != "autospec" {
			continue
		}

		destPath := filepath.Join(destDir, "autospec")
		outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
		if err != nil {
			return "", fmt.Errorf("creating binary file: %w", err)
		}

		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			os.Remove(destPath)
			return "", fmt.Errorf("extracting binary: %w", err)
		}
		outFile.Close()

		return destPath, nil
	}

	return "", fmt.Errorf("autospec binary not found in archive")
}
