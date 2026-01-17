package changelog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultRemoteTimeout is the default timeout for remote changelog fetches.
const DefaultRemoteTimeout = 5 * time.Second

// RemoteChangelogURL is the URL for fetching the remote changelog.
// Can be overridden for testing.
var RemoteChangelogURL = "https://raw.githubusercontent.com/ariel-frischer/autospec/main/internal/changelog/changelog.yaml"

// FetchRemote fetches the changelog from the remote repository.
// Returns the embedded changelog as fallback if remote fetch fails.
// The context can be used to control timeout and cancellation.
func FetchRemote(ctx context.Context) (*Changelog, error) {
	log, err := fetchFromURL(ctx, RemoteChangelogURL)
	if err != nil {
		return nil, fmt.Errorf("fetching remote changelog: %w", err)
	}
	return log, nil
}

// FetchRemoteWithFallback fetches the changelog from the remote repository.
// Falls back to embedded changelog if remote fetch fails.
// Returns the changelog and a boolean indicating if it's from remote.
func FetchRemoteWithFallback(ctx context.Context) (*Changelog, bool, error) {
	log, err := FetchRemote(ctx)
	if err == nil {
		return log, true, nil
	}

	// Fall back to embedded changelog
	embedded, embErr := LoadEmbedded()
	if embErr != nil {
		return nil, false, fmt.Errorf("remote failed (%v) and embedded failed: %w", err, embErr)
	}

	return embedded, false, nil
}

// fetchFromURL fetches and parses a changelog from a URL.
func fetchFromURL(ctx context.Context, url string) (*Changelog, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return LoadFromReader(bytes.NewReader(body))
}

// FetchVersionFromRemote fetches a specific version from the remote changelog.
// Falls back to embedded changelog if remote fetch fails.
func FetchVersionFromRemote(ctx context.Context, version string) (*Version, bool, error) {
	log, isRemote, err := FetchRemoteWithFallback(ctx)
	if err != nil {
		return nil, false, err
	}

	v, err := log.GetVersion(version)
	if err != nil {
		return nil, isRemote, err
	}

	return v, isRemote, nil
}
