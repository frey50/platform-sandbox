# Operational Runbooks

## Runbook 1: Manual Scale-Up/Scale-Down

### When to Use
- Emergency scale-up during an incident
- Preparing for a demo or load test
- Temporarily overriding the downscaler

### Steps

**Scale up manually:**
```bash
./bin/platformctl scale --app sample-app --replicas 5
```

Or with kubectl:
```bash
kubectl scale deployment sample-app --replicas=5 -n sample-app
```

**Scale down manually:**
```bash
./bin/platformctl downscale --app sample-app
```

Or with kubectl:
```bash
kubectl scale deployment sample-app --replicas=0 -n sample-app
```

**Note:** If Argo CD sync runs and you haven't set `ignoreDifferences`, it will revert your manual scale. That's why we configure `ignoreDifferences` on `/spec/replicas`.

---

## Runbook 2: Argo CD Sync Failure

### Symptoms
- App shows "OutOfSync" or "Unknown" in Argo CD UI
- `kubectl get application sample-app -n argocd` shows degraded state
- New commits aren't being applied

### Diagnosis
```bash
# Check application status
kubectl describe application sample-app -n argocd

# Check Argo CD server logs
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-server

# Check application controller logs
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-application-controller
```

### Resolution

**Force a sync:**
```bash
argocd app sync sample-app
```

**Hard refresh (re-evaluate from Git):**
```bash
argocd app get sample-app --hard-refresh
```

**If stuck in a sync loop:**
1. Check if `ignoreDifferences` is properly configured
2. Check if an external tool (downscaler, HPA) is fighting Argo CD
3. Temporarily disable auto-sync:
```bash
argocd app set sample-app --sync-policy none
# Fix the issue
argocd app set sample-app --sync-policy automated --self-heal
```

---

## Runbook 3: Downscaler Not Working

### Symptoms
- Pods aren't scaling down at night
- Downscaler logs show errors
- Replica count stays at baseline despite schedule

### Diagnosis
```bash
# Check downscaler pod status
kubectl get pods -n kube-system -l app=kube-downscaler

# Check downscaler logs
kubectl logs -n kube-system -l app=kube-downscaler --tail=100

# Check deployment annotations
kubectl get deployment sample-app -n sample-app -o jsonpath='{.metadata.annotations}'
```

### Common Causes & Fixes

**1. Missing annotations:**
```bash
# Add downtime annotation
kubectl annotate deployment sample-app -n sample-app   downscaler/downtime="Mon-Fri 19:00-08:00 UTC,Sat-Sun 00:00-24:00 UTC"
```

**2. Wrong timezone:**
Check `apps/sample-app/kube-downscaler-values.yaml`:
```yaml
extraConfig: |
  DEFAULT_TIMEZONE: "UTC"
```

**3. Downscaler doesn't have permissions:**
```bash
kubectl auth can-i list deployments --as=system:serviceaccount:kube-system:kube-downscaler
```

**4. Argo CD is reverting the downscaler:**
Verify `ignoreDifferences` is set in the Application resource:
```bash
kubectl get application sample-app -n argocd -o yaml | grep -A5 ignoreDifferences
```

---

## Runbook 4: Cluster Recovery

### When to Use
- kind cluster was accidentally deleted
- Docker daemon was restarted
- Need to recreate the entire environment

### Steps

```bash
# 1. Delete existing cluster (if partially broken)
kind delete cluster --name platform-sandbox

# 2. Recreate
./cluster-up.sh

# 3. Reinstall Argo CD
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# 4. Reinstall downscaler
helm install kube-downscaler hjacobs/kube-downscaler   --namespace kube-system   --values apps/sample-app/kube-downscaler-values.yaml

# 5. Re-apply the Argo CD Application
kubectl apply -f apps/sample-app-application.yaml -n argocd

# 6. Verify
./bin/platformctl status
```

**Time to recover:** ~3-5 minutes

---

## Runbook 5: Performance Issues

### Symptoms
- App is slow or unresponsive
- High CPU/memory usage
- HPA is constantly scaling

### Diagnosis
```bash
# Check resource usage
kubectl top pods -n sample-app

# Check HPA status
kubectl get hpa -n sample-app

# Check for OOMKilled pods
kubectl get pods -n sample-app -o wide

# Check logs for errors
kubectl logs -n sample-app -l app=sample-app --tail=100
```

### Resolution

**If CPU throttled:**
```bash
# Edit deployment to increase CPU limits
kubectl edit deployment sample-app -n sample-app
# Increase resources.limits.cpu
```

**If HPA is oscillating:**
- Increase ` stabilizationWindowSeconds` in HPA
- Or increase resource requests so HPA has a stable baseline

**If memory pressure:**
```bash
# Check node resources
kubectl describe node <node-name>

# If kind cluster is too small, recreate with more resources:
cat <<EOF | kind create cluster --name platform-sandbox --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        system-reserved: memory=1Gi
- role: worker
  extraMounts:
  - hostPath: /tmp
    containerPath: /data
EOF
```

---

## Runbook 6: Adding a New Application

### Steps

1. **Create app manifests** in a new folder under `apps/`:
```
apps/
  my-new-app/
    namespace.yaml
    deployment.yaml
    service.yaml
    ...
```

2. **Create an Argo CD Application** resource:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-new-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/frey50/platform-sandbox
    targetRevision: main
    path: apps/my-new-app
  destination:
    server: https://kubernetes.default.svc
    namespace: my-new-app
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
  ignoreDifferences:
    - group: apps
      kind: Deployment
      jsonPointers:
        - /spec/replicas
```

3. **Add downscaler annotations** to the Deployment:
```yaml
metadata:
  annotations:
    downscaler/downtime: "Mon-Fri 19:00-08:00 UTC,Sat-Sun 00:00-24:00 UTC"
    downscaler/upscale-replicas: "3"
```

4. **Apply:**
```bash
kubectl apply -f apps/my-new-app-application.yaml -n argocd
```

5. **Verify:**
```bash
./bin/platformctl status
```

---

## Runbook 7: Backup and Disaster Recovery

### What to Back Up

Since this is a GitOps platform, **Git is your backup**. The cluster is ephemeral.

**Critical data in Git:**
- All K8s manifests
- Argo CD Application definitions
- Downscaler configuration
- CLI source code

**Not in Git (ephemeral):**
- Running pod state
- Prometheus metrics history
- Grafana dashboards (if not exported)

### Backup Strategy

1. **Git repository:** Already versioned. Push to remote regularly.
2. **Grafana dashboards:** Export as JSON and commit to `infrastructure/observability/dashboards/`
3. **Argo CD app-of-apps pattern:** Consider an "app-of-apps" that manages all Applications, so one apply restores everything.

### Disaster Recovery Procedure

```bash
# 1. Clone repo (if lost)
git clone https://github.com/frey50/platform-sandbox.git
cd platform-sandbox

# 2. Recreate cluster
make cluster-up

# 3. Restore everything
make deploy-infra
make deploy
make deploy-observability

# 4. Verify
./bin/platformctl status
```

**RTO (Recovery Time Objective):** < 10 minutes  
**RPO (Recovery Point Objective):** 0 (everything is in Git)
