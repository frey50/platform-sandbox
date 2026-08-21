# ------------------------------------------------------------------------------
# Platform Sandbox — Makefile
# Common tasks for local development, deployment, and testing.
# ------------------------------------------------------------------------------

CLUSTER_NAME ?= platform-sandbox
SHELL := /bin/bash

# Colors for pretty output
BLUE  := \033[36m
GREEN := \033[32m
RED   := \033[31m
RESET := \033[0m

.PHONY: help
help: ## Show this help message
	@echo "Platform Sandbox — Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-28s$(RESET) %s\n", $$1, $$2}'

# ------------------------------------------------------------------------------
# Cluster Lifecycle
# ------------------------------------------------------------------------------

.PHONY: cluster-up
cluster-up: ## Create the local kind cluster
	@echo "$(GREEN)==> Creating kind cluster '$(CLUSTER_NAME)'...$(RESET)"
	chmod +x cluster-up.sh
	./cluster-up.sh

.PHONY: cluster-down
cluster-down: ## Delete the local kind cluster
	@echo "$(RED)==> Deleting kind cluster '$(CLUSTER_NAME)'...$(RESET)"
	kind delete cluster --name $(CLUSTER_NAME)

# ------------------------------------------------------------------------------
# Core Platform
# ------------------------------------------------------------------------------

.PHONY: install-argocd
install-argocd: ## Install Argo CD into the cluster
	@echo "$(GREEN)==> Installing Argo CD...$(RESET)"
	kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -n argocd --server-side --force-conflicts -f \
		https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
	kubectl wait --for=condition=available deployment/argocd-server -n argocd --timeout=300s
	@echo "$(GREEN)==> Argo CD admin password:$(RESET)"
	kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d; echo

.PHONY: install-downscaler
install-downscaler: ## Install kube-downscaler via Helm
	@echo "$(GREEN)==> Installing kube-downscaler...$(RESET)"
	helm repo add hjacobs https://hjacobs.github.io/kube-downscaler/ 2>/dev/null || true
	helm repo update
	helm upgrade --install kube-downscaler hjacobs/kube-downscaler \
		--namespace kube-downscaler --create-namespace \
		--values apps/sample-app/kube-downscaler-values.yaml

.PHONY: deploy
deploy: ## Deploy sample-app via Argo CD Application
	@echo "$(GREEN)==> Deploying sample-app via Argo CD...$(RESET)"
	kubectl apply -f apps/sample-app-application.yaml -n argocd

.PHONY: deploy-observability
deploy-observability: ## Deploy Prometheus + Grafana + Loki via Argo CD
	@echo "$(GREEN)==> Deploying observability stack...$(RESET)"
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
	helm repo update
	kubectl apply -f apps/observability-application.yaml -n argocd

# ------------------------------------------------------------------------------
# CLI
# ------------------------------------------------------------------------------

.PHONY: build-cli
build-cli: ## Build the platformctl Go CLI
	@echo "$(GREEN)==> Building platformctl...$(RESET)"
	cd cmd/platformctl && go build -o ../../bin/platformctl .
	@echo "$(GREEN)==> Binary at ./bin/platformctl$(RESET)"

# ------------------------------------------------------------------------------
# Port Forwards
# ------------------------------------------------------------------------------

.PHONY: forward-grafana
forward-grafana: ## Port-forward Grafana to localhost:3000
	@echo "$(GREEN)==> Forwarding Grafana → http://localhost:3000$(RESET)"
	kubectl port-forward svc/platform-grafana 3000:80 -n observability

.PHONY: forward-prometheus
forward-prometheus: ## Port-forward Prometheus to localhost:9090
	@echo "$(GREEN)==> Forwarding Prometheus → http://localhost:9090$(RESET)"
	kubectl port-forward svc/platform-prometheus 9090:9090 -n observability

.PHONY: forward-argocd
forward-argocd: ## Port-forward Argo CD to localhost:8080
	@echo "$(GREEN)==> Forwarding Argo CD → https://localhost:8080$(RESET)"
	kubectl port-forward svc/argocd-server 8080:443 -n argocd

.PHONY: forward-app
forward-app: ## Port-forward sample-app to localhost:8081
	@echo "$(GREEN)==> Forwarding sample-app → http://localhost:8081$(RESET)"
	kubectl port-forward svc/sample-app 8081:80 -n sample-app

# ------------------------------------------------------------------------------
# Testing & Validation
# ------------------------------------------------------------------------------

.PHONY: test
test: ## Run all tests (lint + build + integration smoke test)
	$(MAKE) lint
	$(MAKE) build-cli
	@echo "$(GREEN)==> Running smoke tests...$(RESET)"
	./bin/platformctl verify
	./bin/platformctl status

.PHONY: lint
lint: ## Lint YAML and Go code
	@echo "$(GREEN)==> Linting YAML...$(RESET)"
	yamllint -c .yamllint . 2>/dev/null || echo "yamllint not installed, skipping"
	@echo "$(GREEN)==> Linting Go...$(RESET)"
	cd cmd/platformctl && go vet ./...

.PHONY: validate-manifests
validate-manifests: ## Validate K8s manifests with kubeconform
	@echo "$(GREEN)==> Validating manifests...$(RESET)"
	kubeconform -summary \
		-ignore-filename-pattern ".*values.*" \
		-ignore-filename-pattern "kustomization.yaml" \
		-schema-location default \
		-schema-location "https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json" \
		apps/ infrastructure/

# ------------------------------------------------------------------------------
# Cleanup
# ------------------------------------------------------------------------------

.PHONY: clean
clean: cluster-down ## Tear down everything
	@echo "$(RED)==> Removing build artifacts...$(RESET)"
	rm -rf bin/

# ------------------------------------------------------------------------------
# Observability Helpers
# ------------------------------------------------------------------------------

.PHONY: grafana-password
grafana-password: ## Show the Grafana admin password
	@echo "$(GREEN)Grafana admin password: admin$(RESET)"
	@echo "(Default is 'admin' as set in kube-prometheus-stack-values.yaml)"

.PHONY: logs-loki
logs-loki: ## Show Loki logs
	kubectl logs -n observability -l app.kubernetes.io/name=loki --tail=100

.PHONY: logs-promtail
logs-promtail: ## Show Promtail logs
	kubectl logs -n observability -l app.kubernetes.io/name=promtail --tail=100