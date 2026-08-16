# Architecture Documentation

## Overview

Platform Sandbox is a reference implementation of a GitOps-based Kubernetes platform with built-in cost optimization. It demonstrates how to run dev/staging workloads efficiently while maintaining full observability and operational control.

## System Components

### 1. GitOps Layer: Argo CD

Argo CD watches the Git repository and continuously ensures the cluster state matches the desired state defined in Git.

**Key Configuration:**
- `syncPolicy.automated.selfHeal: true` — Argo CD will revert manual changes
- `syncPolicy.automated.prune: true` — Removes resources deleted from Git
- `syncOptions.CreateNamespace=true` — Auto-creates namespaces
- `ignoreDifferences` on `/spec/replicas` — Allows external actors (downscaler) to modify replica count without triggering sync loops

**Why this matters:** In a pure GitOps setup, any drift from Git is corrected. But for cost optimization, we *want* the downscaler to drift replicas temporarily. `ignoreDifferences` is the escape hatch that makes this pattern work.

### 2. Application Layer: sample-app

A minimal but production-pattern Deployment running `nginxdemos/hello`.

**Production patterns demonstrated:**
- **Resource limits**: Prevents noisy neighbor problems
- **Liveness/Readiness probes**: Ensures traffic only hits healthy pods
- **HPA**: Scales horizontally based on CPU/memory
- **PDB**: Ensures minimum availability during disruptions
- **NetworkPolicy**: Zero-trust networking between pods
- **ServiceAccount + RBAC**: Least-privilege access
- **ConfigMap**: Externalized configuration

### 3. Cost Optimization Layer: kube-downscaler

The kube-downscaler periodically scans Deployments and scales them based on time-based rules.

**How it works:**
1. Watches all namespaces for Deployments with `downscaler/*` annotations
2. Evaluates the current time against the schedule defined in annotations
3. Scales matching Deployments to the specified replica count (usually 0)
4. Restores original replica count when the schedule says "up"

**Integration with Argo CD:**
```yaml
ignoreDifferences:
  - group: apps
    kind: Deployment
    jsonPointers:
      - /spec/replicas
```

This tells Argo CD: "I see the replica count changed, but I won't try to 'fix' it."

### 4. Observability Layer: Prometheus + Grafana + Loki

- **Prometheus**: Scrapes metrics from apps and cluster components
- **Grafana**: Visualizes metrics with pre-built dashboards
- **Loki**: Aggregates logs for centralized debugging
- **ServiceMonitor**: Tells Prometheus which services to scrape

### 5. Operator Layer: platformctl (Go CLI)

A custom CLI that wraps common operational tasks:
- Abstracts kubectl/argocd/helm complexity
- Provides a unified interface for platform operators
- Can be extended with custom business logic

## Data Flow

```
Developer pushes manifest changes to Git
           │
           ▼
    Argo CD detects change
           │
           ▼
    Argo CD applies manifests to cluster
           │
           ├──► Namespace created
           ├──► Deployment created (3 replicas)
           ├──► Service created
           ├──► HPA created
           └──► NetworkPolicy applied
           │
           ▼
    kube-downscaler evaluates schedule
           │
           ├──► Business hours: 3 replicas maintained
           └──► Off-hours: scales to 0 replicas
           │
           ▼
    Prometheus scrapes metrics
           │
           ▼
    Grafana displays dashboards
           │
           ▼
    Operator uses platformctl to inspect/manage
```

## Security Model

| Layer | Control |
|---|---|
| Network | NetworkPolicy restricts pod-to-pod traffic |
| Identity | ServiceAccount per app with minimal RBAC |
| Resources | ResourceQuotas and LimitRanges prevent DOS |
| Secrets | (Placeholder) Sealed Secrets or external vault |
| Ingress | TLS termination at ingress controller |

## Scalability Considerations

While this runs on a local `kind` cluster, the patterns scale to production:

- **Argo CD**: Supports 1000+ apps per instance. Use ApplicationSets for multi-tenant or multi-env setups.
- **kube-downscaler**: Lightweight Python controller. Handles thousands of Deployments.
- **HPA**: Native K8s feature. Works with any metrics server (Prometheus Adapter, KEDA, etc.).

## Failure Scenarios & Mitigations

| Scenario | Impact | Mitigation |
|---|---|---|
| Argo CD is down | No new syncs, but existing workloads run | Argo CD is stateless; restore from Git |
| kube-downscaler is down | No cost savings, apps stay scaled up | Monitor downscaler health; alerts on failure |
| Git repo is unavailable | No new changes, but cluster state persists | Git is the source of truth; cluster keeps running |
| Cluster node failure | Pod rescheduling | PDB ensures graceful disruption; HPA restores capacity |

## Future Enhancements

- [ ] Multi-environment support (dev → staging → prod) via Kustomize or Helm
- [ ] KEDA for event-driven autoscaling (queue-based, cron-based)
- [ ] Sealed Secrets or External Secrets Operator for secret management
- [ ] Argo Rollouts for canary/blue-green deployments
- [ ] Cost allocation tags and showback reports
- [ ] Slack/Teams notifications for sync failures and scaling events
