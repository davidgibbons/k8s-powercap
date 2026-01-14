# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

### Added
- Cron schedule validation: Only single-point in time (no ranges, lists, or step values)
- Multi-schedule support: Agent now parses SCHEDULES_JSON to handle multiple schedules with multiple zones/constraints per schedule

### Fixed
- Fixed read-only `/sys` mount in agent pods - agent can now write to powercap interfaces
- Fixed ServiceAccount reference: DaemonSet now uses default SA instead of non-existent powercapschedule SA
- Fixed config mismatch: Controller and agent now both use SCHEDULES_JSON format
- Added webhook markers: +kubebuilder:webhook annotations for MWH and VWH generation

### Changed
- Agent configuration: Rewritten to parse SCHEDULES_JSON instead of individual env vars (SCHEDULE, POWER_LIMIT, ZONE, CONSTRAINT)
- Agent now supports: Multiple schedules, multiple power limits per schedule

---

## [0.1.0] - 2026-01-13

### Added
- Cron schedule validation: Only single-point in time (no ranges, lists, or step values)
- Multi-schedule support: Agent now parses SCHEDULES_JSON to handle multiple schedules with multiple zones/constraints per schedule

### Fixed
- Fixed read-only `/sys` mount in agent pods - agent can now write to powercap interfaces
- Fixed ServiceAccount reference: DaemonSet now uses default SA instead of non-existent powercapschedule SA
- Fixed config mismatch: Controller and agent now both use SCHEDULES_JSON format

### Changed
- Agent configuration: Rewritten to parse SCHEDULES_JSON instead of individual env vars (SCHEDULE, POWER_LIMIT, ZONE, CONSTRAINT)
- Agent now supports: Multiple schedules, multiple power limits per schedule
