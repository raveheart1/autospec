package update

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloader_DownloadBinary(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		responseCode int
		responseBody string
		wantErr      bool
	}{
		"successful download": {
			responseCode: http.StatusOK,
			responseBody: "fake binary content",
			wantErr:      false,
		},
		"not found": {
			responseCode: http.StatusNotFound,
			responseBody: "",
			wantErr:      true,
		},
		"server error": {
			responseCode: http.StatusInternalServerError,
			responseBody: "",
			wantErr:      true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			downloader := NewDownloader(server.Client())

			path, err := downloader.DownloadBinary(context.Background(), server.URL, nil)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.FileExists(t, path)
			defer os.Remove(path)

			content, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, tt.responseBody, string(content))
		})
	}
}

func TestDownloader_DownloadBinary_WithProgress(t *testing.T) {
	t.Parallel()

	body := "test content for progress tracking"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(body)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	downloader := NewDownloader(server.Client())

	var progressCalls int
	path, err := downloader.DownloadBinary(context.Background(), server.URL, func(current, total int64) {
		progressCalls++
	})

	require.NoError(t, err)
	assert.FileExists(t, path)
	defer os.Remove(path)
	assert.Greater(t, progressCalls, 0)
}

func TestParseChecksum(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content   string
		assetName string
		want      string
		wantErr   bool
	}{
		"valid checksum": {
			content:   "abc123def456  autospec_0.7.0_Linux_x86_64.tar.gz\nfed789  other_file.tar.gz",
			assetName: "autospec_0.7.0_Linux_x86_64.tar.gz",
			want:      "abc123def456",
		},
		"asset not found": {
			content:   "abc123  other_file.tar.gz",
			assetName: "autospec_0.7.0_Linux_x86_64.tar.gz",
			wantErr:   true,
		},
		"empty content": {
			content:   "",
			assetName: "autospec_0.7.0_Linux_x86_64.tar.gz",
			wantErr:   true,
		},
		"with extra whitespace": {
			content:   "  abc123  autospec_0.7.0_Linux_x86_64.tar.gz  \n",
			assetName: "autospec_0.7.0_Linux_x86_64.tar.gz",
			want:      "abc123",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseChecksum(tt.content, tt.assetName)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestVerifyChecksum(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content  string
		checksum string
		wantErr  bool
	}{
		"valid checksum": {
			content:  "test content",
			checksum: "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72",
			wantErr:  false,
		},
		"invalid checksum": {
			content:  "test content",
			checksum: "0000000000000000000000000000000000000000000000000000000000000000",
			wantErr:  true,
		},
		"case insensitive match": {
			content:  "test content",
			checksum: "6AE8A75555209FD6C44157C0AED8016E763FF435A19CF186F76863140143FF72",
			wantErr:  false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpFile, err := os.CreateTemp("", "checksum-test-*")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.content)
			require.NoError(t, err)
			tmpFile.Close()

			err = VerifyChecksum(tmpFile.Name(), tt.checksum)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestFetchChecksum(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		responseCode int
		responseBody string
		assetName    string
		want         string
		wantErr      bool
	}{
		"successful fetch": {
			responseCode: http.StatusOK,
			responseBody: "abc123  autospec_0.7.0_Linux_x86_64.tar.gz\n",
			assetName:    "autospec_0.7.0_Linux_x86_64.tar.gz",
			want:         "abc123",
		},
		"not found": {
			responseCode: http.StatusNotFound,
			responseBody: "",
			assetName:    "autospec_0.7.0_Linux_x86_64.tar.gz",
			wantErr:      true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.responseCode)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			downloader := NewDownloader(server.Client())

			got, err := downloader.FetchChecksum(context.Background(), server.URL, tt.assetName)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewDownloader(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		client *http.Client
	}{
		"with nil client": {
			client: nil,
		},
		"with custom client": {
			client: &http.Client{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d := NewDownloader(tt.client)
			assert.NotNil(t, d)
			assert.NotNil(t, d.httpClient)
		})
	}
}

func TestExtractBinary_NoArchive(t *testing.T) {
	t.Parallel()

	// Create a fake file that's not a valid tar.gz
	tmpFile, err := os.CreateTemp("", "not-a-tar-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("this is not a tar.gz file")
	require.NoError(t, err)
	tmpFile.Close()

	_, err = ExtractBinary(tmpFile.Name(), t.TempDir())
	assert.Error(t, err)
}

func TestExtractBinary_NonexistentFile(t *testing.T) {
	t.Parallel()

	_, err := ExtractBinary("/nonexistent/path/to/file.tar.gz", t.TempDir())
	assert.Error(t, err)
}

func TestProgressWriter(t *testing.T) {
	t.Parallel()

	var updates []struct{ current, total int64 }

	tmpFile, err := os.CreateTemp("", "progress-test-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	pw := &ProgressWriter{
		Writer: tmpFile,
		Total:  100,
		OnUpdate: func(current, total int64) {
			updates = append(updates, struct{ current, total int64 }{current, total})
		},
	}

	data := []byte("hello world")
	n, err := pw.Write(data)

	require.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Greater(t, len(updates), 0)
	assert.Equal(t, int64(len(data)), updates[len(updates)-1].current)
}

func TestInstaller(t *testing.T) {
	// Can't run parallel due to executable path lookup
	// This tests the basic installer creation
	installer, err := NewInstaller()
	require.NoError(t, err)

	assert.NotEmpty(t, installer.GetExecutablePath())
	assert.NotEmpty(t, installer.GetBackupPath())
	assert.Contains(t, installer.GetBackupPath(), ".bak")
}

func TestInstaller_WritePermissionCheck(t *testing.T) {
	t.Parallel()

	// Create a temp directory we have write access to
	tmpDir := t.TempDir()
	tmpBinary := filepath.Join(tmpDir, "test-binary")

	// Create a fake binary
	err := os.WriteFile(tmpBinary, []byte("fake"), 0755)
	require.NoError(t, err)

	installer := &Installer{
		executablePath: tmpBinary,
		backupPath:     tmpBinary + ".bak",
	}

	err = installer.CheckWritePermission()
	assert.NoError(t, err)
}

func TestInstaller_BackupAndRollback(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	originalPath := filepath.Join(tmpDir, "autospec")
	backupPath := originalPath + ".bak"

	// Create original binary
	err := os.WriteFile(originalPath, []byte("original content"), 0755)
	require.NoError(t, err)

	installer := &Installer{
		executablePath: originalPath,
		backupPath:     backupPath,
	}

	// Create backup
	err = installer.CreateBackup()
	require.NoError(t, err)

	assert.NoFileExists(t, originalPath)
	assert.FileExists(t, backupPath)

	// Rollback
	err = installer.Rollback()
	require.NoError(t, err)

	assert.FileExists(t, originalPath)
	assert.NoFileExists(t, backupPath)

	// Verify content is preserved
	content, err := os.ReadFile(originalPath)
	require.NoError(t, err)
	assert.Equal(t, "original content", string(content))
}

func TestInstaller_InstallBinary(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	destPath := filepath.Join(tmpDir, "autospec")
	newBinaryPath := filepath.Join(tmpDir, "new-binary")

	// Create new binary
	err := os.WriteFile(newBinaryPath, []byte("new content"), 0755)
	require.NoError(t, err)

	installer := &Installer{
		executablePath: destPath,
		backupPath:     destPath + ".bak",
	}

	err = installer.InstallBinary(newBinaryPath)
	require.NoError(t, err)

	assert.FileExists(t, destPath)
	assert.NoFileExists(t, newBinaryPath)

	content, err := os.ReadFile(destPath)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(content))
}

func TestInstaller_SetPermissions(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "autospec")

	// Create binary with non-executable permissions
	err := os.WriteFile(binaryPath, []byte("content"), 0644)
	require.NoError(t, err)

	installer := &Installer{
		executablePath: binaryPath,
		backupPath:     binaryPath + ".bak",
	}

	err = installer.SetPermissions()
	require.NoError(t, err)

	info, err := os.Stat(binaryPath)
	require.NoError(t, err)

	// Check that executable bit is set
	assert.True(t, info.Mode()&0111 != 0)
}

func TestInstaller_CleanupBackup(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "autospec")
	backupPath := binaryPath + ".bak"

	// Create backup
	err := os.WriteFile(backupPath, []byte("backup"), 0755)
	require.NoError(t, err)

	installer := &Installer{
		executablePath: binaryPath,
		backupPath:     backupPath,
	}

	err = installer.CleanupBackup()
	require.NoError(t, err)

	assert.NoFileExists(t, backupPath)
}

func TestInstaller_Rollback_NoBackup(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	installer := &Installer{
		executablePath: filepath.Join(tmpDir, "autospec"),
		backupPath:     filepath.Join(tmpDir, "autospec.bak"),
	}

	err := installer.Rollback()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no backup found")
}
