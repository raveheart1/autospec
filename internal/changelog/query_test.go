package changelog

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetVersion(t *testing.T) {
	tests := map[string]struct {
		changelog *Changelog
		version   string
		wantErr   bool
		wantVer   string
	}{
		"exact match": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Added: []string{"A"}}},
				},
			},
			version: "1.0.0",
			wantErr: false,
			wantVer: "1.0.0",
		},
		"with v prefix": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Added: []string{"A"}}},
				},
			},
			version: "v1.0.0",
			wantErr: false,
			wantVer: "1.0.0",
		},
		"uppercase v prefix": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Added: []string{"A"}}},
				},
			},
			version: "V1.0.0",
			wantErr: false,
			wantVer: "1.0.0",
		},
		"unreleased": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "unreleased", Changes: Changes{Added: []string{"A"}}},
				},
			},
			version: "unreleased",
			wantErr: false,
			wantVer: "unreleased",
		},
		"version not found": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Added: []string{"A"}}},
				},
			},
			version: "2.0.0",
			wantErr: true,
		},
		"empty changelog": {
			changelog: &Changelog{
				Project:  "test",
				Versions: []Version{},
			},
			version: "1.0.0",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.changelog.GetVersion(tt.version)
			if tt.wantErr {
				require.Error(t, err)
				var verErr *VersionNotFoundError
				assert.True(t, errors.As(err, &verErr))
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantVer, got.Version)
		})
	}
}

func TestGetUnreleased(t *testing.T) {
	tests := map[string]struct {
		changelog *Changelog
		wantNil   bool
	}{
		"has unreleased": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "unreleased", Changes: Changes{Added: []string{"A"}}},
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Added: []string{"B"}}},
				},
			},
			wantNil: false,
		},
		"no unreleased": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Added: []string{"A"}}},
				},
			},
			wantNil: true,
		},
		"empty changelog": {
			changelog: &Changelog{
				Project:  "test",
				Versions: []Version{},
			},
			wantNil: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.changelog.GetUnreleased()
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.True(t, got.IsUnreleased())
			}
		})
	}
}

func TestListVersions(t *testing.T) {
	tests := map[string]struct {
		changelog *Changelog
		want      []string
	}{
		"multiple versions": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "unreleased", Changes: Changes{Added: []string{"A"}}},
					{Version: "1.1.0", Date: "2026-01-16", Changes: Changes{Added: []string{"B"}}},
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Added: []string{"C"}}},
				},
			},
			want: []string{"unreleased", "1.1.0", "1.0.0"},
		},
		"single version": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Added: []string{"A"}}},
				},
			},
			want: []string{"1.0.0"},
		},
		"empty changelog": {
			changelog: &Changelog{
				Project:  "test",
				Versions: []Version{},
			},
			want: []string{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.changelog.ListVersions()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetLastN(t *testing.T) {
	changelog := &Changelog{
		Project: "test",
		Versions: []Version{
			{
				Version: "unreleased",
				Changes: Changes{Added: []string{"Entry 1", "Entry 2"}},
			},
			{
				Version: "1.0.0",
				Date:    "2026-01-15",
				Changes: Changes{Fixed: []string{"Entry 3"}},
			},
		},
	}

	tests := map[string]struct {
		changelog *Changelog
		n         int
		wantLen   int
	}{
		"get 2 entries": {
			changelog: changelog,
			n:         2,
			wantLen:   2,
		},
		"get more than exists": {
			changelog: changelog,
			n:         10,
			wantLen:   3,
		},
		"get 0 entries": {
			changelog: changelog,
			n:         0,
			wantLen:   0,
		},
		"get negative entries": {
			changelog: changelog,
			n:         -1,
			wantLen:   0,
		},
		"empty changelog": {
			changelog: &Changelog{Project: "test", Versions: []Version{}},
			n:         5,
			wantLen:   0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.changelog.GetLastN(tt.n)
			assert.Len(t, got, tt.wantLen)
		})
	}
}

func TestAllEntries(t *testing.T) {
	tests := map[string]struct {
		changelog *Changelog
		wantLen   int
		wantFirst string
	}{
		"multiple versions": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "unreleased", Changes: Changes{Added: []string{"New feature"}}},
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Fixed: []string{"Bug fix"}}},
				},
			},
			wantLen:   2,
			wantFirst: "New feature",
		},
		"empty changelog": {
			changelog: &Changelog{Project: "test", Versions: []Version{}},
			wantLen:   0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.changelog.AllEntries()
			assert.Len(t, got, tt.wantLen)
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantFirst, got[0].Text)
			}
		})
	}
}

func TestVersionNotFoundError(t *testing.T) {
	tests := map[string]struct {
		err     *VersionNotFoundError
		wantMsg string
	}{
		"with available versions": {
			err: &VersionNotFoundError{
				Version:           "2.0.0",
				AvailableVersions: []string{"1.0.0", "1.1.0"},
			},
			wantMsg: `version "2.0.0" not found (available: 1.0.0, 1.1.0)`,
		},
		"empty available versions": {
			err: &VersionNotFoundError{
				Version:           "1.0.0",
				AvailableVersions: []string{},
			},
			wantMsg: `version "1.0.0" not found (available: )`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.wantMsg, tt.err.Error())
		})
	}
}

func TestGetVersionCount(t *testing.T) {
	tests := map[string]struct {
		changelog *Changelog
		want      int
	}{
		"multiple versions": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "unreleased", Changes: Changes{Added: []string{"A"}}},
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Added: []string{"B"}}},
				},
			},
			want: 2,
		},
		"empty changelog": {
			changelog: &Changelog{Project: "test", Versions: []Version{}},
			want:      0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.changelog.GetVersionCount())
		})
	}
}

func TestGetEntryCount(t *testing.T) {
	tests := map[string]struct {
		changelog *Changelog
		want      int
	}{
		"multiple entries across versions": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "unreleased", Changes: Changes{Added: []string{"A", "B"}}},
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Fixed: []string{"C"}}},
				},
			},
			want: 3,
		},
		"empty changelog": {
			changelog: &Changelog{Project: "test", Versions: []Version{}},
			want:      0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.changelog.GetEntryCount())
		})
	}
}

func TestHasUnreleased(t *testing.T) {
	tests := map[string]struct {
		changelog *Changelog
		want      bool
	}{
		"has unreleased": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "unreleased", Changes: Changes{Added: []string{"A"}}},
				},
			},
			want: true,
		},
		"no unreleased": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Added: []string{"A"}}},
				},
			},
			want: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.changelog.HasUnreleased())
		})
	}
}

func TestGetLatestRelease(t *testing.T) {
	tests := map[string]struct {
		changelog *Changelog
		wantNil   bool
		wantVer   string
	}{
		"with releases": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "unreleased", Changes: Changes{Added: []string{"A"}}},
					{Version: "1.1.0", Date: "2026-01-16", Changes: Changes{Added: []string{"B"}}},
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Added: []string{"C"}}},
				},
			},
			wantNil: false,
			wantVer: "1.1.0",
		},
		"only unreleased": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "unreleased", Changes: Changes{Added: []string{"A"}}},
				},
			},
			wantNil: true,
		},
		"no unreleased": {
			changelog: &Changelog{
				Project: "test",
				Versions: []Version{
					{Version: "1.0.0", Date: "2026-01-15", Changes: Changes{Added: []string{"A"}}},
				},
			},
			wantNil: false,
			wantVer: "1.0.0",
		},
		"empty changelog": {
			changelog: &Changelog{Project: "test", Versions: []Version{}},
			wantNil:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.changelog.GetLatestRelease()
			if tt.wantNil {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.wantVer, got.Version)
			}
		})
	}
}
