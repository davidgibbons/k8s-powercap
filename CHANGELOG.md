## [1.1.3](https://github.com/davidgibbons/k8s-powercap/compare/v1.1.2...v1.1.3) (2026-01-15)


### Bug Fixes

* **main.go:** add entryID to map for efficient schedule name retrieval ([a0ee087](https://github.com/davidgibbons/k8s-powercap/commit/a0ee087bfbd13ec42df59542ba09f9ed86128052))

## [1.1.2](https://github.com/davidgibbons/k8s-powercap/compare/v1.1.1...v1.1.2) (2026-01-15)


### Bug Fixes

* **main.go:** remove unnecessary variable reassignment for rule in loop iteration ([c4ffe80](https://github.com/davidgibbons/k8s-powercap/commit/c4ffe804ffc7d5e5ce1ce7475218a4af8b66f1e0))

## [1.1.1](https://github.com/davidgibbons/k8s-powercap/compare/v1.1.0...v1.1.1) (2026-01-15)


### Bug Fixes

* **powercap.go:** add constraint parameter to ValidatePowercapPath function ([e4eda44](https://github.com/davidgibbons/k8s-powercap/commit/e4eda44f613fc3b7d9959f0b19a7b8f159f76e0f))

# [1.1.0](https://github.com/davidgibbons/k8s-powercap/compare/v1.0.1...v1.1.0) (2026-01-15)


### Bug Fixes

* **e2e_test.go:** add wait for controller-manager pod to be ready in e2e tests ([ac30618](https://github.com/davidgibbons/k8s-powercap/commit/ac306184d9cb987b3479070cd4cc8d168d949e4f))


### Features

* **.github/workflows:** add support for workflow_dispatch event to workflows/build-agent.yaml and workflows/build-controller.yaml ([40c7620](https://github.com/davidgibbons/k8s-powercap/commit/40c76206d6d9a672f37d4a90bc9bf70c207fda0f))
* **controller:** implement timezone-aware scheduling ([7578d7e](https://github.com/davidgibbons/k8s-powercap/commit/7578d7e9160cd93ee178b1baa2d314cd64efe1c6))

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
