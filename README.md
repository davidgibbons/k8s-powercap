# k8s-powercap

Kubernetes operator for controlling powercap settings on nodes based on cron schedules.

## Quickstart (Helm)

```bash
helm repo add k8s-powercap https://github.com/davidgibbons/k8s-powercap/releases/download/helm-charts
helm repo update
helm install k8s-powercap k8s-powercap/k8s-powercap \
  --namespace k8s-powercap-system --create-namespace
```

Example values override (optional):

```yaml
controller:
  image:
    repository: ghcr.io/davidgibbons/k8s-powercap
    tag: latest
agent:
  image:
    repository: ghcr.io/davidgibbons/k8s-powercap-agent
    tag: latest
```

## Images

- Controller: `ghcr.io/davidgibbons/k8s-powercap:<tag>`
- Agent: `ghcr.io/davidgibbons/k8s-powercap-agent:<tag>`

## Overview

This operator allows you to dynamically manage power limits on your Kubernetes nodes using the Linux powercap interface. It deploys DaemonSets that run on selected nodes and apply power limits according to a cron-based schedule.

## Features

- **Cron-based scheduling**: Use standard cron expressions to define when power limits should be applied
- **Node targeting**: Use node selectors to target specific groups of nodes
- **Automatic DaemonSet management**: The operator creates and manages DaemonSets on your behalf
- **Rolling updates**: Configuration changes trigger rolling updates of DaemonSet pods
- **Status tracking**: Monitor when power limits were last applied and when they'll be applied next
- **Suspend support**: Temporarily disable power limit changes without deleting resources

## Architecture

```
PowercapSchedule CR → Controller → DaemonSet (Agent Pods)
                                        ↓
                                    /sys/class/powercap
                                        ↓
                                    Apply Power Limits
```

1. **PowercapSchedule CRD**: Define your power schedules and node targets
2. **Controller**: Reconciles CR changes and manages DaemonSets
3. **DaemonSet**: Runs privileged agent pods on selected nodes
4. **Agent**: Uses cron to apply power limits to `/sys/class/powercap`

## Prerequisites

- Kubernetes cluster with privileged pod support
- Nodes with Linux powercap support (typically Intel RAPL)
- go version v1.24.6+
- Container build tool (Docker or Podman), if building images
- kubectl version compatible with your cluster

## Getting Started

### Build and Push Images

```bash
# Build controller image
make docker-build IMG=<some-registry>/k8s-powercap:tag
make docker-push IMG=<some-registry>/k8s-powercap:tag

# Build agent image
make docker-build-agent AGENT_IMG=<some-registry>/k8s-powercap-agent:tag
make docker-push-agent AGENT_IMG=<some-registry>/k8s-powercap-agent:tag
```

### Deploy to Cluster

```bash
# Install or upgrade via Helm (CRDs are included in the chart)
helm upgrade --install k8s-powercap k8s-powercap/k8s-powercap \
  --namespace k8s-powercap-system --create-namespace \
  --set controller.image.repository=<some-registry>/k8s-powercap \
  --set controller.image.tag=tag \
  --set agent.image.repository=<some-registry>/k8s-powercap-agent \
  --set agent.image.tag=tag
```

Update the agent image reference if needed:

```bash
helm upgrade --install k8s-powercap k8s-powercap/k8s-powercap \
  --namespace k8s-powercap-system \
  --set agent.image.repository=<your-agent-image-repo> \
  --set agent.image.tag=<your-agent-image-tag>
```

### Create a PowercapSchedule

```yaml
apiVersion: powercap.k8s.io/v1
kind: PowercapSchedule
metadata:
  name: workday-powercap
  namespace: default
spec:
  # Cron expression: 9 AM on Monday
  schedule: "0 9 * * 1"

  # Timezone (defaults to UTC)
  timeZone: "America/Los_Angeles"

  # Power limit in microwatts (65 Watts = 65000000 microwatts)
  powerLimit: 65000000

  # Which powercap zone to control (e.g., intel-rapl:0)
  zone: "intel-rapl:0"

  # Which constraint within zone (default: constraint_0)
  constraint: "constraint_0"

  # Node selector to target specific nodes
  nodeSelector:
    hardware: cpu

  # Temporarily suspend scheduling
  suspend: false
```

Apply the manifest:

```bash
kubectl apply -f examples/powercap_v1_powercapschedule.yaml
```

### Verify Status

```bash
kubectl get powercapschedules -o wide
kubectl describe powercapschedule workday-powercap
```

Status includes:
- `lastScheduleTime`: When the power limit was last applied
- `nextScheduleTime`: When the next schedule will trigger
- `currentPowerLimit`: The current power limit value
- `conditions`: Current state of the resource

### Check DaemonSets

The operator creates a DaemonSet for each PowercapSchedule:

```bash
kubectl get daemonsets -n default
kubectl logs -n default -l app.kubernetes.io/name=powercapschedule -c agent
```

## Configuration

### Powercap Zones

List available powercap zones on a node:

```bash
# Run on a node
ls /sys/class/powercap

# Typical output:
# intel-rapl:0  intel-rapl:1
```

Each zone contains constraints:

```bash
ls /sys/class/powercap/intel-rapl:0
# constraint_0  constraint_1  name  ...
```

### Power Limit Units

Power limits are specified in **microwatts (µW)**:

| Watts | Microwatts |
|-------|-------------|
| 10 W  | 10,000,000  |
| 65 W  | 65,000,000  |
| 100 W | 100,000,000 |
| 250 W | 250,000,000 |

### Cron Schedule Format

Schedules must represent a single point in time. Each field accepts either a specific number or `*` (wildcard). Ranges, lists, and step values are not supported.

```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6) (Sunday to Saturday)
│ │ │ │ │
* * * * *
```

Examples:
- `0 9 * * 1` - 9 AM on Monday
- `0 0 * * *` - Midnight every day
- `30 14 1 * *` - 2:30 PM on the 1st of every month
- `* * * * *` - Every minute

## Troubleshooting

### DaemonSet Pods Not Starting

Check events:

```bash
kubectl describe daemonset -n default workday-powercap-daemon
```

Common issues:
- **Node not matching selector**: Verify node labels with `kubectl get nodes --show-labels`
- **Privileged access**: Ensure pods have required permissions
- **Powercap not available**: Node may not support powercap/RAPL

### Power Limits Not Applied

Check agent logs:

```bash
kubectl logs -n default <pod-name> -c agent
```

Verify powercap path exists on the node:

```bash
# On the node
ls /sys/class/powercap/intel-rapl:0/constraint_0
cat /sys/class/powercap/intel-rapl:0/constraint_0_power_limit_uw
```

### CRD Validation Errors

Check `kubectl describe` for validation error details. Common issues:
- Invalid cron expression
- Power limit out of range (must be > 0)
- Missing required fields

## To Uninstall

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete powercapschedules --all
```

**Uninstall the Helm release (removes controller, agent, and CRDs):**

```sh
helm uninstall k8s-powercap -n k8s-powercap-system
```

## Project Distribution

Use the Helm chart for the recommended install flow.
See `chart/k8s-powercap/README.md` for install commands and values.

## Contributing
Contributions are welcome. Please open an issue to discuss larger changes before sending a PR.
For local development, run `make test` and ensure any new behavior is covered by tests.

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

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
