/*
Copyright 2026 Derek Gibbons.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package powercap

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestSetPowerLimit(t *testing.T) {
	tests := []struct {
		name                string
		setupMockFilesystem func() (string, func())
		zone                string
		constraint          string
		powerLimit          int64
		expectedError       bool
	}{
		{
			name: "successfully set power limit",
			setupMockFilesystem: func() (string, func()) {
				tmpDir := t.TempDir()
				zoneDir := filepath.Join(tmpDir, "intel-rapl:0")
				if err := os.MkdirAll(zoneDir, 0755); err != nil {
					t.Fatalf("failed to create mock zone: %v", err)
				}
				limitFile := filepath.Join(zoneDir, "constraint_0_power_limit_uw")
				if err := os.WriteFile(limitFile, []byte("100000000"), 0644); err != nil {
					t.Fatalf("failed to create mock limit file: %v", err)
				}
				return tmpDir, func() {}
			},
			zone:       "intel-rapl:0",
			constraint: "constraint_0",
			powerLimit: 65000000,
		},
		{
			name: "use default constraint when empty",
			setupMockFilesystem: func() (string, func()) {
				tmpDir := t.TempDir()
				zoneDir := filepath.Join(tmpDir, "intel-rapl:0")
				if err := os.MkdirAll(zoneDir, 0755); err != nil {
					t.Fatalf("failed to create mock zone: %v", err)
				}
				limitFile := filepath.Join(zoneDir, "constraint_0_power_limit_uw")
				if err := os.WriteFile(limitFile, []byte("100000000"), 0644); err != nil {
					t.Fatalf("failed to create mock limit file: %v", err)
				}
				return tmpDir, func() {}
			},
			zone:       "intel-rapl:0",
			constraint: "",
			powerLimit: 65000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := tt.setupMockFilesystem()
			defer cleanup()

			logger := zaptest.NewLogger(t).Sugar()
			pm := NewPowercapManager(logger)

			oldSysfsPath := sysfsPowercapPath
			defer func() { sysfsPowercapPath = oldSysfsPath }()
			sysfsPowercapPath = tmpDir

			err := pm.SetPowerLimit(tt.zone, tt.constraint, tt.powerLimit)

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectedError {
				constraintName := tt.constraint
				if constraintName == "" {
					constraintName = "constraint_0"
				}
				limitFile := filepath.Join(tmpDir, tt.zone, constraintName+"_power_limit_uw")
				data, err := os.ReadFile(limitFile)
				if err != nil {
					t.Errorf("failed to read limit file: %v", err)
				}

				expectedStr := strconv.FormatInt(tt.powerLimit, 10)
				actualStr := string(data)
				if actualStr != expectedStr {
					t.Errorf("expected limit %q, got %q", expectedStr, actualStr)
				}
			}
		})
	}
}

func TestReadCurrentPowerLimit(t *testing.T) {
	tests := []struct {
		name                string
		setupMockFilesystem func() (string, func())
		zone                string
		constraint          string
		expectedLimit       int64
		expectedError       bool
	}{
		{
			name: "successfully read power limit",
			setupMockFilesystem: func() (string, func()) {
				tmpDir := t.TempDir()
				zoneDir := filepath.Join(tmpDir, "intel-rapl:0")
				if err := os.MkdirAll(zoneDir, 0755); err != nil {
					t.Fatalf("failed to create mock zone: %v", err)
				}
				limitFile := filepath.Join(zoneDir, "constraint_0_power_limit_uw")
				if err := os.WriteFile(limitFile, []byte("65000000"), 0644); err != nil {
					t.Fatalf("failed to create mock limit file: %v", err)
				}
				return tmpDir, func() {}
			},
			zone:          "intel-rapl:0",
			constraint:    "constraint_0",
			expectedLimit: 65000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := tt.setupMockFilesystem()
			defer cleanup()

			logger := zaptest.NewLogger(t).Sugar()
			pm := NewPowercapManager(logger)

			oldSysfsPath := sysfsPowercapPath
			defer func() { sysfsPowercapPath = oldSysfsPath }()
			sysfsPowercapPath = tmpDir

			constraintName := tt.constraint
			if constraintName == "" {
				constraintName = "constraint_0"
			}
			limitFile := filepath.Join(tmpDir, tt.zone, constraintName+"_power_limit_uw")
			if err := os.WriteFile(limitFile, []byte(strconv.FormatInt(tt.expectedLimit, 10)), 0644); err != nil {
				t.Fatalf("failed to write limit file: %v", err)
			}

			limit, err := pm.ReadCurrentPowerLimit(tt.zone, tt.constraint)

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectedError && limit != tt.expectedLimit {
				t.Errorf("expected limit %d, got %d", tt.expectedLimit, limit)
			}
		})
	}
}

func TestValidatePowercapPath(t *testing.T) {
	tests := []struct {
		name                string
		setupMockFilesystem func() (string, func())
		zone                string
		expectedError       bool
	}{
		{
			name: "valid RAPL zone",
			setupMockFilesystem: func() (string, func()) {
				tmpDir := t.TempDir()
				zoneDir := filepath.Join(tmpDir, "intel-rapl:0")
				constraintDir := filepath.Join(zoneDir, "constraint_0")
				if err := os.MkdirAll(constraintDir, 0755); err != nil {
					t.Fatalf("failed to create mock filesystem: %v", err)
				}
				return tmpDir, func() {}
			},
			zone:          "intel-rapl:0",
			expectedError: false,
		},
		{
			name: "zone does not exist",
			setupMockFilesystem: func() (string, func()) {
				tmpDir := t.TempDir()
				zoneDir := filepath.Join(tmpDir, "intel-rapl:0")
				if err := os.MkdirAll(zoneDir, 0755); err != nil {
					t.Fatalf("failed to create mock filesystem: %v", err)
				}
				return tmpDir, func() {}
			},
			zone:          "intel-rapl:99",
			expectedError: true,
		},
		{
			name: "non-RAPL zone but valid",
			setupMockFilesystem: func() (string, func()) {
				tmpDir := t.TempDir()
				zoneDir := filepath.Join(tmpDir, "other-zone")
				constraintDir := filepath.Join(zoneDir, "constraint_0")
				if err := os.MkdirAll(constraintDir, 0755); err != nil {
					t.Fatalf("failed to create mock filesystem: %v", err)
				}
				return tmpDir, func() {}
			},
			zone:          "other-zone",
			expectedError: false,
		},
		{
			name: "zone is a file not directory",
			setupMockFilesystem: func() (string, func()) {
				tmpDir := t.TempDir()
				zoneFile := filepath.Join(tmpDir, "intel-rapl:0")
				if err := os.WriteFile(zoneFile, []byte("not a directory"), 0644); err != nil {
					t.Fatalf("failed to create mock file: %v", err)
				}
				return tmpDir, func() {}
			},
			zone:          "intel-rapl:0",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := tt.setupMockFilesystem()
			defer cleanup()

			logger := zaptest.NewLogger(t).Sugar()
			pm := NewPowercapManager(logger)

			oldSysfsPath := sysfsPowercapPath
			defer func() { sysfsPowercapPath = oldSysfsPath }()
			sysfsPowercapPath = tmpDir

			err := pm.ValidatePowercapPath(tt.zone)

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestListAvailableZones(t *testing.T) {
	tests := []struct {
		name                string
		setupMockFilesystem func() (string, func())
		expectedZones       []string
	}{
		{
			name: "multiple zones available",
			setupMockFilesystem: func() (string, func()) {
				tmpDir := t.TempDir()
				for _, zone := range []string{"intel-rapl:0", "intel-rapl:1", "other-zone"} {
					zoneDir := filepath.Join(tmpDir, zone)
					if err := os.MkdirAll(zoneDir, 0755); err != nil {
						t.Fatalf("failed to create mock zone: %v", err)
					}
				}
				file := filepath.Join(tmpDir, "not-a-zone.txt")
				if err := os.WriteFile(file, []byte("content"), 0644); err != nil {
					t.Fatalf("failed to create mock file: %v", err)
				}
				return tmpDir, func() {}
			},
			expectedZones: []string{"intel-rapl:0", "intel-rapl:1", "other-zone"},
		},
		{
			name: "no zones available",
			setupMockFilesystem: func() (string, func()) {
				tmpDir := t.TempDir()
				return tmpDir, func() {}
			},
			expectedZones: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := tt.setupMockFilesystem()
			defer cleanup()

			logger := zaptest.NewLogger(t).Sugar()
			pm := NewPowercapManager(logger)

			oldSysfsPath := sysfsPowercapPath
			defer func() { sysfsPowercapPath = oldSysfsPath }()
			sysfsPowercapPath = tmpDir

			zones, err := pm.ListAvailableZones()

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(zones) != len(tt.expectedZones) {
				t.Errorf("expected %d zones, got %d", len(tt.expectedZones), len(zones))
			}
		})
	}
}

func TestGetMaxPowerLimit(t *testing.T) {
	tests := []struct {
		name                string
		setupMockFilesystem func() (string, func())
		zone                string
		constraint          string
		expectedMax         int64
		expectedError       bool
	}{
		{
			name: "successfully read max power limit",
			setupMockFilesystem: func() (string, func()) {
				tmpDir := t.TempDir()
				zoneDir := filepath.Join(tmpDir, "intel-rapl:0")
				if err := os.MkdirAll(zoneDir, 0755); err != nil {
					t.Fatalf("failed to create mock zone: %v", err)
				}
				maxFile := filepath.Join(zoneDir, "constraint_0_max_power_uw")
				if err := os.WriteFile(maxFile, []byte("100000000"), 0644); err != nil {
					t.Fatalf("failed to create mock max file: %v", err)
				}
				return tmpDir, func() {}
			},
			zone:        "intel-rapl:0",
			constraint:  "constraint_0",
			expectedMax: 100000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := tt.setupMockFilesystem()
			defer cleanup()

			logger := zaptest.NewLogger(t).Sugar()
			pm := NewPowercapManager(logger)

			oldSysfsPath := sysfsPowercapPath
			defer func() { sysfsPowercapPath = oldSysfsPath }()
			sysfsPowercapPath = tmpDir

			constraintName := tt.constraint
			if constraintName == "" {
				constraintName = "constraint_0"
			}
			maxFile := filepath.Join(tmpDir, tt.zone, constraintName+"_max_power_uw")
			if err := os.WriteFile(maxFile, []byte(strconv.FormatInt(tt.expectedMax, 10)), 0644); err != nil {
				t.Fatalf("failed to write max file: %v", err)
			}

			max, err := pm.GetMaxPowerLimit(tt.zone, tt.constraint)

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.expectedError && max != tt.expectedMax {
				t.Errorf("expected max %d, got %d", tt.expectedMax, max)
			}
		})
	}
}
