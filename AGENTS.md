# Repository Guidelines

## Project Structure & Module Organization
- `cmd/`: entry points for the controller (`cmd/main.go`) and agent (`cmd/agent/main.go`).
- `api/`: CRD API definitions and generated types.
- `internal/`: controller logic and supporting packages not intended for external use.
- `pkg/`: shared library code used across components.
- `config/`: Kustomize manifests and CRD YAMLs.
- `chart/`: Helm chart for deployment.
- `test/`: unit/integration tests and `test/e2e` for end-to-end tests.
- `examples/`: sample CRs and manifests.
- `hack/`: tooling scripts and boilerplate headers.

## Build, Test, and Development Commands
- `make build`: build the controller binary to `bin/manager`.
- `make build-agent`: build the agent binary to `bin/agent`.
- `make run`: run the controller locally against your kubeconfig.
- `make test`: run unit/integration tests using envtest (excludes e2e).
- `make test-e2e`: run Ginkgo e2e tests with Kind (uses `KIND_CLUSTER`); set `CERT_MANAGER_INSTALL_SKIP=true` to skip cert-manager install.
- `make lint`: run `golangci-lint` with repo rules.
- `make docker-build` / `make docker-build-agent`: build controller/agent images (set `IMG`/`AGENT_IMG`).

## Coding Style & Naming Conventions
- Go formatting is enforced via `gofmt` and `goimports` (see `.golangci.yml`).
- Prefer standard Go naming: exported `CamelCase`, unexported `camelCase`, file names `snake_case.go`.
- Keep CRD/API changes in `api/` and regenerate with `make manifests generate`.

## Testing Guidelines
- Unit/integration: `go test` via `make test` (envtest assets required).
- E2E: Ginkgo/Gomega under `test/e2e` and run with `make test-e2e`.
- Test files follow Go conventions: `*_test.go`.

## Commit & Pull Request Guidelines
- Commit messages follow Conventional Commits when possible (e.g., `chore(workflows): ...`).
- Use a Gitflow-style workflow: create `feature/*` branches for new work and `hotfix/*` branches for urgent fixes; merge via PRs.
- PRs should include a brief summary, testing notes (commands run), and any relevant configuration or CRD changes.
- For user-facing changes, update `README.md` or Helm chart docs as needed.

## Security & Configuration Tips
- The agent requires privileged access to `/sys/class/powercap`; validate cluster policies before deploy.
- Ensure `kubectl`, `kind`, and a container runtime (Docker/Podman) are installed for local/e2e workflows.
