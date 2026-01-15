/*
Copyright 2026 Derek Gibbons.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with License.
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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

var sysfsPowercapPath = "/sys/class/powercap"

const (
	defaultConstraint = "constraint_0"
	powerLimitFile    = defaultConstraint + "_power_limit_uw"
	raplPrefix        = "intel-rapl"
)

type PowercapManager struct {
	logger *zap.SugaredLogger
}

func NewPowercapManager(logger *zap.SugaredLogger) *PowercapManager {
	return &PowercapManager{
		logger: logger,
	}
}

func (p *PowercapManager) SetPowerLimit(zone, constraint string, powerLimitMicrowatts int64) error {
	if zone == "" {
		return fmt.Errorf("zone cannot be empty")
	}

	constraintName := constraint
	if constraintName == "" {
		constraintName = defaultConstraint
	}

	limitFile := fmt.Sprintf("%s_power_limit_uw", constraintName)
	zonePath := filepath.Join(sysfsPowercapPath, zone, limitFile)

	p.logger.Infow("Setting power limit",
		"zone", zone,
		"constraint", constraintName,
		"limit_uw", powerLimitMicrowatts,
		"limit_w", float64(powerLimitMicrowatts)/1000000,
		"path", zonePath,
	)

	limitStr := strconv.FormatInt(powerLimitMicrowatts, 10)

	if err := os.WriteFile(zonePath, []byte(limitStr), 0644); err != nil {
		return fmt.Errorf("failed to write power limit to %s: %w", zonePath, err)
	}

	current, err := p.ReadCurrentPowerLimit(zone, constraint)
	if err == nil {
		p.logger.Infow("Power limit set successfully",
			"requested_uw", powerLimitMicrowatts,
			"current_uw", current,
		)
	}

	return nil
}

func (p *PowercapManager) ReadCurrentPowerLimit(zone, constraint string) (int64, error) {
	if zone == "" {
		return 0, fmt.Errorf("zone cannot be empty")
	}

	constraintName := constraint
	if constraintName == "" {
		constraintName = defaultConstraint
	}

	limitFile := fmt.Sprintf("%s_power_limit_uw", constraintName)
	zonePath := filepath.Join(sysfsPowercapPath, zone, limitFile)

	data, err := os.ReadFile(zonePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read power limit from %s: %w", zonePath, err)
	}

	limit, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse power limit: %w", err)
	}

	return limit, nil
}

func (p *PowercapManager) ValidatePowercapPath(zone, constraint string) error {
	if zone == "" {
		return fmt.Errorf("zone cannot be empty")
	}

	zonePath := filepath.Join(sysfsPowercapPath, zone)

	info, err := os.Stat(zonePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("powercap zone %s does not exist at %s", zone, zonePath)
		}
		return fmt.Errorf("failed to access powercap zone %s: %w", zone, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("powercap zone %s is not a directory", zone)
	}

	if !p.isRAPLZone(zone) {
		p.logger.Warnw("Zone is not a RAPL zone",
			"zone", zone,
			"note", "This may still work if it's a valid powercap device",
		)
	}

	constraintName := constraint
	if constraintName == "" {
		constraintName = defaultConstraint
	}

	limitFile := filepath.Join(zonePath, fmt.Sprintf("%s_power_limit_uw", constraintName))
	if _, err := os.Stat(limitFile); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to access %s in zone %s: %w", constraintName, zone, err)
	}

	constraintPath := filepath.Join(zonePath, constraintName)
	if _, err := os.Stat(constraintPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s does not exist in zone %s", constraintName, zone)
		}
		return fmt.Errorf("failed to access %s in zone %s: %w", constraintName, zone, err)
	}

	return nil
}

func (p *PowercapManager) isRAPLZone(zone string) bool {
	return strings.HasPrefix(zone, raplPrefix)
}

func (p *PowercapManager) ListAvailableZones() ([]string, error) {
	entries, err := os.ReadDir(sysfsPowercapPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read powercap directory: %w", err)
	}

	var zones []string
	for _, entry := range entries {
		if entry.IsDir() {
			zones = append(zones, entry.Name())
		}
	}

	return zones, nil
}

func (p *PowercapManager) GetMaxPowerLimit(zone, constraint string) (int64, error) {
	if zone == "" {
		return 0, fmt.Errorf("zone cannot be empty")
	}

	constraintName := constraint
	if constraintName == "" {
		constraintName = defaultConstraint
	}

	maxFile := fmt.Sprintf("%s_max_power_uw", constraintName)
	zonePath := filepath.Join(sysfsPowercapPath, zone, maxFile)

	data, err := os.ReadFile(zonePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read max power limit from %s: %w", zonePath, err)
	}

	max, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse max power limit: %w", err)
	}

	return max, nil
}
