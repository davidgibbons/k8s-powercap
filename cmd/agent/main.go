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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	powercap "github.com/davidgibbons/k8s-powercap/pkg/powercap"
)

const (
	defaultTimeZone = "UTC"
)

type PowerLimitEntry struct {
	Zone       string `json:"zone"`
	Constraint string `json:"constraint"`
	PowerLimit int64  `json:"powerLimit"`
}

type PowercapRule struct {
	Name        string            `json:"name"`
	Schedule    string            `json:"schedule"`
	PowerLimits []PowerLimitEntry `json:"powerLimits"`
}

type AgentConfig struct {
	Schedules []PowercapRule
	TimeZone  string
	Namespace string
	PodName   string
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	loggerConfig := zap.NewProductionConfig()
	loggerConfig.EncoderConfig.TimeKey = "timestamp"
	loggerConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	loggerConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)

	logger, err := loggerConfig.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = logger.Sync() }()
	sugar := logger.Sugar()

	config, err := loadConfig()
	if err != nil {
		sugar.Fatalw("Failed to load configuration", "error", err)
	}

	sugar.Infow("Starting powercap agent",
		"timezone", config.TimeZone,
		"schedule_count", len(config.Schedules),
	)

	pm := powercap.NewPowercapManager(sugar)

	if zones, err := pm.ListAvailableZones(); err != nil {
		sugar.Warnw("Failed to list available powercap zones", "error", err)
	} else {
		sugar.Infow("Available powercap zones", "zones", zones)
	}

	for _, rule := range config.Schedules {
		for _, limit := range rule.PowerLimits {
			if err := pm.ValidatePowercapPath(limit.Zone, limit.Constraint); err != nil {
				sugar.Fatalw("Failed to validate powercap path",
					"zone", limit.Zone,
					"error", err,
				)
			}
		}
	}

	c := cron.New(cron.WithParser(cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)))

	loc, err := time.LoadLocation(config.TimeZone)
	if err != nil {
		sugar.Warnw("Invalid timezone, using UTC", "timezone", config.TimeZone, "error", err)
		loc = time.UTC
	}

	for _, rule := range config.Schedules {
		_, err = c.AddFunc(rule.Schedule, func() {
			sugar.Infow("Cron schedule triggered", "schedule_name", rule.Name)
			for _, limit := range rule.PowerLimits {
				if err := applyPowerLimit(pm, limit); err != nil {
					sugar.Errorw("Failed to apply power limit",
						"schedule_name", rule.Name,
						"zone", limit.Zone,
						"constraint", limit.Constraint,
						"error", err,
					)
				} else {
					sugar.Infow("Applied power limit",
						"schedule_name", rule.Name,
						"zone", limit.Zone,
						"constraint", limit.Constraint,
						"power_limit_w", float64(limit.PowerLimit)/1000000,
					)
				}
			}
		})
		if err != nil {
			sugar.Fatalw("Failed to add cron schedule", "schedule_name", rule.Name, "error", err)
		}
		sugar.Infow("Registered schedule", "name", rule.Name, "cron", rule.Schedule)
	}

	c.Start()

	for _, entry := range c.Entries() {
		sugar.Infow("Next scheduled run", "at", entry.Next.In(loc).Format(time.RFC3339))
	}

	sugar.Info("Agent started successfully")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		sugar.Infow("Received signal, shutting down", "signal", sig)
	case <-ctx.Done():
		sugar.Info("Context cancelled, shutting down")
	}

	c.Stop()

	sugar.Info("Agent stopped")
}

func loadConfig() (*AgentConfig, error) {
	config := &AgentConfig{
		TimeZone:  getEnvWithDefault("TIMEZONE", defaultTimeZone),
		Namespace: os.Getenv("NAMESPACE"),
		PodName:   os.Getenv("POD_NAME"),
	}

	schedulesJSON := os.Getenv("SCHEDULES_JSON")
	if schedulesJSON == "" {
		return nil, fmt.Errorf("SCHEDULES_JSON environment variable is required")
	}

	if err := json.Unmarshal([]byte(schedulesJSON), &config.Schedules); err != nil {
		return nil, fmt.Errorf("failed to parse SCHEDULES_JSON: %w", err)
	}

	if len(config.Schedules) == 0 {
		return nil, fmt.Errorf("at least one schedule must be defined in SCHEDULES_JSON")
	}

	for _, rule := range config.Schedules {
		if rule.Name == "" {
			return nil, fmt.Errorf("schedule name is required")
		}
		if rule.Schedule == "" {
			return nil, fmt.Errorf("schedule cron expression is required for %q", rule.Name)
		}
		if len(rule.PowerLimits) == 0 {
			return nil, fmt.Errorf("at least one power limit is required for schedule %q", rule.Name)
		}
		for _, limit := range rule.PowerLimits {
			if limit.Zone == "" {
				return nil, fmt.Errorf("zone is required in power limit for schedule %q", rule.Name)
			}
			if limit.Constraint == "" {
				return nil, fmt.Errorf("constraint is required in power limit for schedule %q", rule.Name)
			}
			if limit.PowerLimit <= 0 {
				return nil, fmt.Errorf("power limit must be positive for schedule %q", rule.Name)
			}
		}
	}

	return config, nil
}

func applyPowerLimit(pm *powercap.PowercapManager, limit PowerLimitEntry) error {
	max, err := pm.GetMaxPowerLimit(limit.Zone, limit.Constraint)
	if err != nil {
		return fmt.Errorf("failed to get max power limit: %w", err)
	}

	if limit.PowerLimit > max {
		return fmt.Errorf("power limit %d exceeds max %d", limit.PowerLimit, max)
	}

	return pm.SetPowerLimit(limit.Zone, limit.Constraint, limit.PowerLimit)
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
