// Package progress_test tests progress type definitions, stage status enums, and StageInfo validation.
// Related: internal/progress/types.go
// Tags: progress, types, validation, stage-status, stage-info
package progress_test

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/progress"
)

// TestStageStatus_String tests the String() method of StageStatus enum
func TestStageStatus_String(t *testing.T) {
	tests := map[string]struct {
		status progress.StageStatus
		want   string
	}{
		"pending status": {
			status: progress.StagePending,
			want:   "pending",
		},
		"in_progress status": {
			status: progress.StageInProgress,
			want:   "in_progress",
		},
		"completed status": {
			status: progress.StageCompleted,
			want:   "completed",
		},
		"failed status": {
			status: progress.StageFailed,
			want:   "failed",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.status.String()
			if got != tt.want {
				t.Errorf("StageStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestStageInfo_Validate tests all validation rules for StageInfo
func TestStageInfo_Validate(t *testing.T) {
	tests := map[string]struct {
		stage   progress.StageInfo
		wantErr bool
		errMsg  string
	}{
		"valid stage info": {
			stage: progress.StageInfo{
				Name:        "specify",
				Number:      1,
				TotalStages: 3,
				Status:      progress.StageInProgress,
				RetryCount:  0,
				MaxRetries:  3,
			},
			wantErr: false,
		},
		"empty name": {
			stage: progress.StageInfo{
				Name:        "",
				Number:      1,
				TotalStages: 3,
				Status:      progress.StageInProgress,
			},
			wantErr: true,
			errMsg:  "stage name cannot be empty",
		},
		"number less than or equal to zero": {
			stage: progress.StageInfo{
				Name:        "test",
				Number:      0,
				TotalStages: 3,
			},
			wantErr: true,
			errMsg:  "stage number must be > 0",
		},
		"negative number": {
			stage: progress.StageInfo{
				Name:        "test",
				Number:      -1,
				TotalStages: 3,
			},
			wantErr: true,
			errMsg:  "stage number must be > 0",
		},
		"number exceeds total stages": {
			stage: progress.StageInfo{
				Name:        "test",
				Number:      4,
				TotalStages: 3,
			},
			wantErr: true,
			errMsg:  "stage number cannot exceed total stages",
		},
		"total stages less than or equal to zero": {
			stage: progress.StageInfo{
				Name:        "test",
				Number:      1,
				TotalStages: 0,
			},
			wantErr: true,
			errMsg:  "total stages must be > 0",
		},
		"negative total stages": {
			stage: progress.StageInfo{
				Name:        "test",
				Number:      1,
				TotalStages: -1,
			},
			wantErr: true,
			errMsg:  "total stages must be > 0",
		},
		"negative retry count": {
			stage: progress.StageInfo{
				Name:        "test",
				Number:      1,
				TotalStages: 3,
				RetryCount:  -1,
			},
			wantErr: true,
			errMsg:  "retry count cannot be negative",
		},
		"negative max retries": {
			stage: progress.StageInfo{
				Name:        "test",
				Number:      1,
				TotalStages: 3,
				RetryCount:  0,
				MaxRetries:  -1,
			},
			wantErr: true,
			errMsg:  "max retries cannot be negative",
		},
		"valid with retry attempt": {
			stage: progress.StageInfo{
				Name:        "plan",
				Number:      2,
				TotalStages: 4,
				Status:      progress.StageInProgress,
				RetryCount:  1,
				MaxRetries:  3,
			},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.stage.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("StageInfo.Validate() error = nil, want error containing %q", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("StageInfo.Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("StageInfo.Validate() unexpected error = %v", err)
				}
			}
		})
	}
}
