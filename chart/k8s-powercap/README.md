# Helm Chart for k8s-powercap

A Helm chart for deploying k8s-powercap Kubernetes operator, which manages power usage via powercap and schedules.

## Installation

```bash
helm repo add k8s-powercap https://github.com/davidgibbons/k8s-powercap/releases/download/helm-charts
helm repo update
helm install k8s-powercap k8s-powercap/k8s-powercap --namespace k8s-powercap-system --create-namespace
```

## Images

- Controller: `ghcr.io/davidgibbons/k8s-powercap:<tag>`
- Agent: `ghcr.io/davidgibbons/k8s-powercap-agent:<tag>`

## Values

See [values.yaml](values.yaml) for available configuration options.

## GitHub Actions

The project includes GitHub Actions for automated container builds:

- **Build Controller Image**: `.github/workflows/build-controller.yaml` - Builds the controller image on push
- **Build Agent Image**: `.github/workflows/build-agent.yaml` - Builds the agent image on push

Both workflows push images to `ghcr.io/davidgibbons/k8s-powercap:latest` with proper tagging based on Git version.
