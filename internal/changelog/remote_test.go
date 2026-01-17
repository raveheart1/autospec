package changelog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchFromURL(t *testing.T) {
	validYAML := `project: autospec
versions:
  - version: "0.7.0"
    date: "2025-01-01"
    changes:
      added:
        - Test feature
`

	tests := map[string]struct {
		handler      http.HandlerFunc
		wantErr      bool
		wantErrMsg   string
		wantProject  string
		wantVersions int
	}{
		"successful fetch": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(validYAML))
			},
			wantErr:      false,
			wantProject:  "autospec",
			wantVersions: 1,
		},
		"server error": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:    true,
			wantErrMsg: "unexpected status code: 500",
		},
		"not found": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr:    true,
			wantErrMsg: "unexpected status code: 404",
		},
		"invalid YAML": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("invalid: [yaml"))
			},
			wantErr:    true,
			wantErrMsg: "parsing changelog",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			ctx := context.Background()
			log, err := fetchFromURL(ctx, server.URL)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantProject, log.Project)
			assert.Len(t, log.Versions, tt.wantVersions)
		})
	}
}

func TestFetchFromURL_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := fetchFromURL(ctx, server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestFetchFromURL_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := fetchFromURL(ctx, server.URL)
	require.Error(t, err)
}

func TestFetchRemote_IntegrationWithGlobal(t *testing.T) {
	// This test validates FetchRemote uses RemoteChangelogURL correctly
	// Not parallel to avoid race on global
	validYAML := `project: autospec
versions:
  - version: "0.7.0"
    date: "2025-01-01"
    changes:
      added:
        - Test feature
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(validYAML))
	}))
	defer server.Close()

	originalURL := RemoteChangelogURL
	RemoteChangelogURL = server.URL
	defer func() { RemoteChangelogURL = originalURL }()

	ctx := context.Background()
	log, err := FetchRemote(ctx)
	require.NoError(t, err)
	assert.Equal(t, "autospec", log.Project)
}

func TestFetchRemoteWithFallback_Success(t *testing.T) {
	// Not parallel to avoid race on global
	validYAML := `project: autospec
versions:
  - version: "0.8.0"
    date: "2025-02-01"
    changes:
      added:
        - Remote feature
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(validYAML))
	}))
	defer server.Close()

	originalURL := RemoteChangelogURL
	RemoteChangelogURL = server.URL
	defer func() { RemoteChangelogURL = originalURL }()

	ctx := context.Background()
	log, isRemote, err := FetchRemoteWithFallback(ctx)

	require.NoError(t, err)
	assert.True(t, isRemote)
	assert.NotNil(t, log)

	_, verErr := log.GetVersion("0.8.0")
	assert.NoError(t, verErr)
}

func TestFetchRemoteWithFallback_FallsBack(t *testing.T) {
	// Not parallel to avoid race on global
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalURL := RemoteChangelogURL
	RemoteChangelogURL = server.URL
	defer func() { RemoteChangelogURL = originalURL }()

	ctx := context.Background()
	log, isRemote, err := FetchRemoteWithFallback(ctx)

	require.NoError(t, err)
	assert.False(t, isRemote)
	assert.NotNil(t, log)
}

func TestFetchRemoteWithFallback_TimeoutFallback(t *testing.T) {
	// Not parallel to avoid race on global
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	originalURL := RemoteChangelogURL
	RemoteChangelogURL = server.URL
	defer func() { RemoteChangelogURL = originalURL }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	log, isRemote, err := FetchRemoteWithFallback(ctx)

	require.NoError(t, err)
	assert.False(t, isRemote)
	assert.NotNil(t, log)
}

func TestFetchVersionFromRemote(t *testing.T) {
	// Not parallel to avoid race on global
	validYAML := `project: autospec
versions:
  - version: "0.9.0"
    date: "2025-03-01"
    changes:
      added:
        - Version 0.9.0 feature
  - version: "0.8.0"
    date: "2025-02-01"
    changes:
      fixed:
        - Bug fix
`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(validYAML))
	}))
	defer server.Close()

	originalURL := RemoteChangelogURL
	RemoteChangelogURL = server.URL
	defer func() { RemoteChangelogURL = originalURL }()

	tests := map[string]struct {
		version    string
		wantErr    bool
		wantRemote bool
	}{
		"existing version from remote": {
			version:    "0.9.0",
			wantErr:    false,
			wantRemote: true,
		},
		"another version from remote": {
			version:    "0.8.0",
			wantErr:    false,
			wantRemote: true,
		},
		"version with v prefix": {
			version:    "v0.9.0",
			wantErr:    false,
			wantRemote: true,
		},
		"non-existent version": {
			version:    "99.99.99",
			wantErr:    true,
			wantRemote: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			v, isRemote, err := FetchVersionFromRemote(ctx, tt.version)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantRemote, isRemote)
			assert.NotNil(t, v)
		})
	}
}
