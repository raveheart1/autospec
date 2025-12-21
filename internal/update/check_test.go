package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChecker_CheckForUpdate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		currentVersion  string
		responseCode    int
		responseBody    string
		wantAvailable   bool
		wantLatest      string
		wantDownloadURL bool
		wantErr         bool
	}{
		"update available": {
			currentVersion: "v0.6.0",
			responseCode:   http.StatusOK,
			responseBody: `{
				"tag_name": "v0.7.0",
				"published_at": "2025-12-20T00:00:00Z",
				"assets": [
					{"name": "autospec_0.7.0_Linux_x86_64.tar.gz", "browser_download_url": "https://example.com/download.tar.gz"},
					{"name": "checksums.txt", "browser_download_url": "https://example.com/checksums.txt"}
				]
			}`,
			wantAvailable:   true,
			wantLatest:      "v0.7.0",
			wantDownloadURL: true,
		},
		"already up to date": {
			currentVersion: "v0.7.0",
			responseCode:   http.StatusOK,
			responseBody: `{
				"tag_name": "v0.7.0",
				"published_at": "2025-12-20T00:00:00Z",
				"assets": []
			}`,
			wantAvailable: false,
			wantLatest:    "v0.7.0",
		},
		"current newer than latest": {
			currentVersion: "v0.8.0",
			responseCode:   http.StatusOK,
			responseBody: `{
				"tag_name": "v0.7.0",
				"published_at": "2025-12-20T00:00:00Z",
				"assets": []
			}`,
			wantAvailable: false,
			wantLatest:    "v0.7.0",
		},
		"dev build skips check": {
			currentVersion: "dev",
			responseCode:   http.StatusOK,
			responseBody:   `{}`,
			wantAvailable:  false,
		},
		"rate limit error": {
			currentVersion: "v0.6.0",
			responseCode:   http.StatusForbidden,
			responseBody:   `{"message": "rate limit exceeded"}`,
			wantErr:        true,
		},
		"not found error": {
			currentVersion: "v0.6.0",
			responseCode:   http.StatusNotFound,
			responseBody:   `{"message": "not found"}`,
			wantErr:        true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))
				w.WriteHeader(tt.responseCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			checker := NewChecker(5 * time.Second)
			checker.apiURL = server.URL

			result, err := checker.CheckForUpdate(context.Background(), tt.currentVersion)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantAvailable, result.UpdateAvailable)
			if tt.wantLatest != "" {
				assert.Equal(t, tt.wantLatest, result.LatestVersion)
			}
			if tt.wantDownloadURL {
				assert.NotEmpty(t, result.DownloadURL)
			}
		})
	}
}

func TestChecker_Timeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tag_name": "v0.7.0"}`))
	}))
	defer server.Close()

	checker := NewChecker(10 * time.Millisecond)
	checker.apiURL = server.URL

	ctx := context.Background()
	_, err := checker.CheckForUpdate(ctx, "v0.6.0")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "executing request")
}

func TestChecker_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tag_name": "v0.7.0"}`))
	}))
	defer server.Close()

	checker := NewChecker(5 * time.Second)
	checker.apiURL = server.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := checker.CheckForUpdate(ctx, "v0.6.0")

	assert.Error(t, err)
}

func TestBuildAssetName(t *testing.T) {
	t.Parallel()

	// This tests the buildAssetName function indirectly
	// The actual output depends on runtime.GOOS and runtime.GOARCH
	// We just verify it produces a valid format
	name := buildAssetName("v0.7.0")
	assert.Contains(t, name, "autospec_0.7.0_")
	assert.Contains(t, name, ".tar.gz")
}
