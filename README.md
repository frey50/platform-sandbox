# Platform Sandbox

> A GitOps-powered Kubernetes cost optimization platform for dev/staging environments. Automatically scales non-production workloads to zero during off-hours, cutting cloud costs by 70%+ while maintaining full GitOps traceability via Argo CD.

---

## 🎯 What Problem This Solves

Dev and staging environments often run 24/7 with fixed replica counts, burning cloud budget on idle compute. This platform demonstrates a production-ready approach to:

- **Deploy apps via GitOps** (Argo CD) with automated sync and self-healing
- **Automatically downscale workloads** to zero replicas during nights/weekends using `kube-downscaler`
- **Preserve Argo CD sync compatibility** via `ignoreDifferences` so the downscaler's changes don't trigger endless sync loops
- **Provide a custom CLI** (`platformctl`) for operators to inspect, scale, and manage the platform
- **Run everything locally** with `kind` for demos, testing, and portfolio showcases

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        GitHub Repo                            │
│  ┌─────────────────┐  ┌──────────────────────────────────┐  │
│  │  Argo CD App    │  │  K8s Manifests (Deployment,     │  │
│  │  (apps/sample-  │  │  Service, HPA, NetworkPolicy,   │  │
│  │   app-application│  │  RBAC, Ingress, etc.)          │  │
│  │   .yaml)         │  └──────────────────────────────────┘  │
│  └────────┬────────┘                                        │
└───────────┼─────────────────────────────────────────────────┘
            │  GitOps Sync (automated, self-healing)
            ▼
┌─────────────────────────────────────────────────────────────┐
│                     Argo CD (argocd ns)                     │
│              Watches repo → applies to cluster                │
└──────────────────────────┬──────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                  Kubernetes Cluster (kind)                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │ sample-app   │  │ kube-        │  │ Prometheus +     │  │
│  │ (Deployment) │  │ downscaler   │  │ Grafana          │  │
│  │ 3 replicas   │  │ (scales to 0 │  │ (observability)  │  │
│  │ by day)      │  │  at night)   │  │                  │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │ Ingress      │  │ HPA          │  │ platformctl CLI  │  │
│  │ (nginx)      │  │ (autoscaling)│  │ (operator tool)  │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

| Decision | Rationale |
|---|---|
| **Argo CD `ignoreDifferences`** on `/spec/replicas` | Prevents Argo CD from fighting the downscaler. Git declares the "desired baseline" (3 replicas), but the downscaler can temporarily override it without triggering a sync loop. |
| **GitOps for downscaler config** | Downscaler values live in Git, versioned and auditable alongside app manifests. |
| **Custom Go CLI** | Provides a unified operator interface instead of chaining `kubectl` + `argocd` + `helm` commands. |
| **Local `kind` cluster** | Zero-cost demo environment that runs on any laptop. Identical patterns apply to EKS/GKE/AKS. |

---

## 📁 Repository Layout

```
platform-sandbox/
├── LICENSE                           # Apache 2.0
├── README.md                         # You are here
├── Makefile                          # Common tasks (cluster-up, deploy, test, build-cli)
├── cluster-up.sh                     # Bootstrap a local kind cluster
├── .github/
│   └── workflows/
│       ├── ci.yaml                   # Lint, validate, and test manifests
│       └── release-cli.yaml          # Build and release platformctl binaries
├── cmd/
│   └── platformctl/                  # Go CLI source code
│       ├── main.go
│       ├── cmd/
│       │   ├── root.go
│       │   ├── status.go
│       │   ├── scale.go
│       │   ├── downscale.go
│       │   ├── up.go
│       │   ├── verify.go
│       │   └── logs.go
│       └── pkg/
│           ├── k8s/                  # Kubernetes client helpers
│           ├── argocd/               # Argo CD API helpers
│           └── downscaler/           # Downscaler integration
├── apps/
│   ├── sample-app-application.yaml   # Argo CD Application resource
│   └── sample-app/
│       ├── namespace.yaml            # Namespace + labels
│       ├── deployment.yaml           # App Deployment (nginxdemos/hello)
│       ├── service.yaml              # ClusterIP Service
│       ├── hpa.yaml                  # HorizontalPodAutoscaler
│       ├── pdb.yaml                  # PodDisruptionBudget
│       ├── networkpolicy.yaml        # NetworkPolicy (least privilege)
│       ├── serviceaccount.yaml       # SA + RBAC
│       ├── configmap.yaml            # App configuration
│       ├── ingress.yaml              # Ingress rules
│       ├── servicemonitor.yaml       # Prometheus ServiceMonitor
│       └── kube-downscaler-values.yaml  # Downscaler Helm values
├── infrastructure/
│   ├── argocd/                       # Argo CD installation manifests
│   ├── downscaler/                   # kube-downscaler Helm values + manifests
│   └── observability/                # Prometheus + Grafana + Loki
└── docs/
    ├── architecture.md               # Detailed architecture docs
    ├── setup.md                      # Step-by-step setup guide
    ├── runbooks.md                   # Operational runbooks
    └── troubleshooting.md            # Common issues and fixes
```

---

## 🚀 Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- [Helm](https://helm.sh/docs/intro/install/) (v3+)
- [Go](https://go.dev/dl/) 1.22+ (for building `platformctl`)

### 1. Bootstrap the Cluster

```bash
git clone https://github.com/frey50/platform-sandbox.git
cd platform-sandbox
chmod +x cluster-up.sh
./cluster-up.sh
```

Override the cluster name if desired:
```bash
CLUSTER_NAME=my-cluster ./cluster-up.sh
```

### 2. Install Argo CD

```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
# Wait for pods
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=argocd-server -n argocd --timeout=120s
```

### 3. Install kube-downscaler

```bash
helm repo add hjacobs https://hjacobs.github.io/kube-downscaler/
helm repo update
helm install kube-downscaler hjacobs/kube-downscaler   --namespace kube-system   --values apps/sample-app/kube-downscaler-values.yaml
```

### 4. Deploy the Sample App via Argo CD

```bash
kubectl apply -f apps/sample-app-application.yaml -n argocd
```

Argo CD will:
- Create the `sample-app` namespace
- Sync all manifests from `apps/sample-app/`
- Start 3 replicas of `nginxdemos/hello`

### 5. Verify Everything

```bash
# Check Argo CD sync status
kubectl get application sample-app -n argocd

# Check pods
kubectl get pods -n sample-app

# Check downscaler logs
kubectl logs -n kube-system -l app=kube-downscaler

# Access the app (port-forward or via Ingress)
kubectl port-forward svc/sample-app 8080:80 -n sample-app
# Open http://localhost:8080
```

### 6. Build & Use the CLI

```bash
make build-cli
./bin/platformctl status
./bin/platformctl scale --app sample-app --replicas 5
./bin/platformctl downscale --app sample-app
```

---

## 💰 Cost Optimization in Action

By default, the sample app runs **3 replicas during business hours** and **scales to 0 overnight and weekends**.

| Scenario | Replicas | Estimated Monthly Cost (t3.medium equiv) |
|---|---|---|
| Always-on (no downscaler) | 3 | ~$45 |
| With downscaler (biz hours only) | 3 → 0 | ~$15 |
| **Savings** | | **~67%** |

*Actual savings depend on your cloud provider, instance types, and schedule configuration.*

---

## 🛠️ Makefile Commands

```bash
make cluster-up          # Create kind cluster
make cluster-down        # Delete kind cluster
make deploy              # Apply Argo CD app + manifests
make deploy-observability # Install Prometheus + Grafana
make build-cli           # Build platformctl binary
make test                # Run CI tests locally
make lint                # Lint YAML and manifests
make clean               # Remove cluster + artifacts
```

---

## 📊 Observability

After running `make deploy-observability`:

| Tool | Access | Credentials |
|---|---|---|
| **Grafana** | `kubectl port-forward svc/grafana 3000:3000 -n observability` | `admin` / `admin` |
| **Prometheus** | `kubectl port-forward svc/prometheus 9090:9090 -n observability` | N/A |
| **Argo CD UI** | `kubectl port-forward svc/argocd-server 8080:443 -n argocd` | `admin` / get from `argocd-initial-admin-secret` |

Pre-built dashboards include:
- Pod count over time (with downscaler events)
- Argo CD sync health
- App request latency and throughput

---

## 🔐 Security Considerations

- **RBAC**: Each app runs under its own `ServiceAccount` with minimal permissions
- **NetworkPolicy**: Ingress/egress restricted to necessary ports and namespaces only
- **No secrets in Git**: Sensitive values are injected via Sealed Secrets or external secret managers (placeholder for production)
- **Resource limits**: All containers have CPU/memory requests and limits set

---

## 🤝 Contributing

1. Fork the repo
2. Create a feature branch: `git checkout -b feat/amazing-thing`
3. Commit your changes: `git commit -m 'Add amazing thing'`
4. Push to the branch: `git push origin feat/amazing-thing`
5. Open a Pull Request

See [docs/setup.md](docs/setup.md) for development environment setup.

---

## 📜 License

[Apache 2.0](LICENSE)

---

## 🙋 FAQ

**Q: Why `ignoreDifferences` on replicas instead of letting Argo CD manage them?**  
A: If Argo CD manages replicas, it will constantly fight the downscaler. `ignoreDifferences` tells Argo CD: "Git is the source of truth for everything *except* replica count."

**Q: Can I use this on EKS/GKE/AKS instead of kind?**  
A: Absolutely. Replace `cluster-up.sh` with your cloud cluster creation, and the rest of the manifests apply identically.

**Q: How do I change the downscale schedule?**  
A: Edit `apps/sample-app/kube-downscaler-values.yaml` and add annotations to your Deployments. See [docs/runbooks.md](docs/runbooks.md).

**Q: What if I need an app to stay up 24/7?**  
A: Add the annotation `downscaler/exclude: "true"` to that Deployment. The downscaler will skip it.

---

*Built with ☕, Go, and way too many YAML files.*
