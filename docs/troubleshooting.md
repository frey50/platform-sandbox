# Troubleshooting Guide

## Common Issues and Solutions

### Issue 1: `cluster-up.sh` fails with "kind is not installed"

**Error:**
```
ERROR: kind is not installed. Install it: https://kind.sigs.k8s.io/docs/user/quick-start/#installation
```

**Solution:**
```bash
# macOS
brew install kind

# Linux
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.22.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# Verify
kind --version
```

---

### Issue 2: Docker daemon not running

**Error:**
```
ERROR: docker is not installed or not running.
```

**Solution:**
```bash
# macOS
open -a Docker

# Linux
sudo systemctl start docker

# Verify
docker ps
```

---

### Issue 3: kind cluster creation hangs

**Symptoms:**
- `kind create cluster` hangs indefinitely
- Docker shows high CPU/memory usage

**Causes & Solutions:**

**A. Insufficient Docker resources:**
```bash
# Check Docker Desktop settings
# Increase memory to at least 4GB, CPUs to 2+
```

**B. Existing cluster name conflict:**
```bash
# Check existing clusters
kind get clusters

# Delete if exists
kind delete cluster --name platform-sandbox

# Recreate
./cluster-up.sh
```

**C. Kind config issue:**
```bash
# Try with minimal config
cat <<EOF | kind create cluster --name platform-sandbox --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
EOF
```

---

### Issue 4: kubectl context not set correctly

**Symptoms:**
- `kubectl get nodes` shows wrong cluster or connection refused
- Commands target a different cluster

**Solution:**
```bash
# List contexts
kubectl config get-contexts

# Switch to platform-sandbox
kubectl config use-context kind-platform-sandbox

# Verify
kubectl cluster-info
```

---

### Issue 5: Argo CD pods stuck in Pending

**Symptoms:**
```bash
kubectl get pods -n argocd
# Shows pods in Pending state
```

**Diagnosis:**
```bash
kubectl describe pod <pod-name> -n argocd
# Look for: Insufficient memory, Insufficient cpu, PersistentVolumeClaim pending
```

**Solutions:**

**A. Resource constraints:**
Increase Docker Desktop resources (8GB RAM, 4 CPUs recommended for full stack).

**B. Recreate cluster with more resources:**
```bash
kind delete cluster --name platform-sandbox

# Create with resource reservations
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
        system-reserved: memory=2Gi,cpu=1
        kube-reserved: memory=1Gi,cpu=500m
- role: worker
- role: worker
EOF
```

---

### Issue 6: Argo CD Application stays OutOfSync

**Symptoms:**
- App status shows "OutOfSync" in Argo CD UI
- Sync attempts fail

**Diagnosis:**
```bash
# Get detailed status
argocd app get sample-app

# Or via kubectl
kubectl describe application sample-app -n argocd
```

**Common Causes:**

**A. Git repo not accessible:**
```bash
# Check repo URL in Application spec
kubectl get application sample-app -n argocd -o yaml | grep repoURL

# Verify connectivity
curl -I https://github.com/frey50/platform-sandbox
```

**B. Missing namespace:**
Ensure `CreateNamespace=true` is in syncOptions:
```yaml
syncOptions:
  - CreateNamespace=true
```

**C. CRDs not installed:**
If using custom resources, ensure their CRDs are installed first.

**D. Resource conflict:**
```bash
# Check for existing resources
kubectl get all -n sample-app

# If resources were created manually, delete them and let Argo CD recreate
kubectl delete namespace sample-app
# Then trigger sync
argocd app sync sample-app
```

---

### Issue 7: Downscaler not scaling deployments

**Symptoms:**
- Pods remain at full replica count during scheduled downtime
- Downscaler logs show no activity

**Diagnosis:**
```bash
# Check downscaler is running
kubectl get pods -n kube-system -l app=kube-downscaler

# Check logs
kubectl logs -n kube-system -l app=kube-downscaler --tail=50

# Check deployment annotations
kubectl get deployment sample-app -n sample-app -o jsonpath='{.metadata.annotations}' | jq .
```

**Solutions:**

**A. Missing or incorrect annotations:**
```bash
# Add proper annotations
kubectl annotate deployment sample-app -n sample-app   downscaler/downtime="Mon-Fri 19:00-08:00 UTC,Sat-Sun 00:00-24:00 UTC"   --overwrite
```

**B. Downscaler can't see the namespace:**
By default, downscaler watches all namespaces. If restricted:
```bash
# Check downscaler args
kubectl get deployment kube-downscaler -n kube-system -o yaml | grep -A20 args

# Ensure --include-resources or --exclude-resources isn't blocking your app
```

**C. Timezone mismatch:**
```bash
# Check downscaler timezone config
kubectl get configmap -n kube-system | grep downscaler

# Verify your annotation timezone matches
# If downscaler uses UTC but your annotation says "Europe/Berlin", it won't match
```

**D. Argo CD fighting the downscaler:**
```bash
# Verify ignoreDifferences
kubectl get application sample-app -n argocd -o jsonpath='{.spec.ignoreDifferences}'

# Should show:
# [{"group":"apps","kind":"Deployment","jsonPointers":["/spec/replicas"]}]
```

---

### Issue 8: platformctl build fails

**Error:**
```
go: module requires go 1.22
```

**Solution:**
```bash
# Check Go version
go version

# Install Go 1.22+
# macOS
brew install go@1.22

# Linux
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

---

### Issue 9: HPA shows "unable to get metrics"

**Symptoms:**
```bash
kubectl describe hpa sample-app -n sample-app
# Shows: unable to get metrics for resource cpu
```

**Solution:**
```bash
# Install metrics-server (required for HPA)
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# For kind, you need to patch metrics-server to skip TLS verification
kubectl patch deployment metrics-server -n kube-system --type='json' -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"}]'

# Wait for metrics-server to be ready
kubectl wait --for=condition=ready pod -l k8s-app=metrics-server -n kube-system --timeout=120s

# Verify HPA can see metrics
kubectl get hpa -n sample-app
```

---

### Issue 10: Ingress not working

**Symptoms:**
- `kubectl get ingress` shows no address
- Can't access app via Ingress hostname

**Solution:**
```bash
# Install ingress-nginx for kind
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

# Wait for controller
kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=controller -n ingress-nginx --timeout=120s

# Add to /etc/hosts
echo "127.0.0.1 sample-app.local" | sudo tee -a /etc/hosts

# Access
curl http://sample-app.local
```

---

### Issue 11: Prometheus not scraping sample-app metrics

**Symptoms:**
- No targets for sample-app in Prometheus
- Grafana dashboard shows "No data"

**Solution:**
```bash
# Check ServiceMonitor exists
kubectl get servicemonitor -n sample-app

# Check Prometheus pod logs
kubectl logs -n observability -l app.kubernetes.io/name=prometheus

# Verify ServiceMonitor selector matches Service labels
kubectl get service sample-app -n sample-app --show-labels
kubectl get servicemonitor sample-app -n sample-app -o yaml | grep -A10 selector

# Check if Prometheus has the right role binding
kubectl auth can-i get servicemonitors --as=system:serviceaccount:observability:prometheus-kube-prometheus-stack-prometheus -n sample-app
```

---

## Getting More Help

1. **Check logs:**
```bash
# App logs
kubectl logs -n sample-app -l app=sample-app --tail=100

# Argo CD logs
kubectl logs -n argocd -l app.kubernetes.io/name=argocd-server --tail=100

# Downscaler logs
kubectl logs -n kube-system -l app=kube-downscaler --tail=100
```

2. **Check events:**
```bash
kubectl get events --all-namespaces --sort-by='.lastTimestamp' | tail -50
```

3. **Describe resources:**
```bash
kubectl describe deployment sample-app -n sample-app
kubectl describe pod <pod-name> -n sample-app
kubectl describe node <node-name>
```

4. **Run platformctl verify:**
```bash
./bin/platformctl verify
```
