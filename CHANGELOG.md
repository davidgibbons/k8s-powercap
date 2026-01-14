## [1.0.1](https://github.com/davidgibbons/k8s-powercap/compare/v1.0.0...v1.0.1) (2026-01-14)


### Bug Fixes

* **Makefile:** add support for extra helm arguments in deploy command ([70c6b53](https://github.com/davidgibbons/k8s-powercap/commit/70c6b53f50ac5cd1a8d89a45172cc75db55bb318))

# 1.0.0 (2026-01-14)


### Bug Fixes

* **.releaserc.json:** update GITHUB_REPOSITORY to process.env.GITHUB_REPOSITORY for environment variable support ([2dd1c93](https://github.com/davidgibbons/k8s-powercap/commit/2dd1c93660b4bb9ba8fd4220371806109b4f7048))
* **e2e_test.go:** update kubectl command to use correct secret name for cert-manager verification ([8cc122e](https://github.com/davidgibbons/k8s-powercap/commit/8cc122e2784c179b2fba80138409e2d001caf19e))
* **powercapschedule_types.go:** update cron expression documentation to allow day/month/dow ranges ([eb4d798](https://github.com/davidgibbons/k8s-powercap/commit/eb4d798608a76453cb7a24936b1384e97566069b))

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
