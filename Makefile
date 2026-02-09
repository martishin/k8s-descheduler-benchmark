.PHONY: setup minikube-up minikube-delete preflight bench-maintenance bench-maintenance-descheduler descheduler-logs monitoring-up dashboards-import descheduler-servicemonitor grafana-port-forward prometheus-port-forward cleanup fmt tidy format test test-ci help

DESCHBENCH := go run ./cmd/deschedbench
PODS ?= 60
PROFILE ?= low-node-utilization

minikube-up: ## Start a 4-node minikube cluster (3 workers) with control-plane metrics enabled
	minikube start -p deschedbench --kubernetes-version=v1.32.0 --cpus=2 --memory=4096 --nodes=4 \
	  --extra-config=kubelet.max-pods=250 \
	  --extra-config=controller-manager.bind-address=0.0.0.0 \
	  --extra-config=scheduler.bind-address=0.0.0.0 \
	  --extra-config=etcd.listen-metrics-urls=http://0.0.0.0:2381
	kubectl config use-context deschedbench
	kubectl taint nodes deschedbench node-role.kubernetes.io/control-plane=:NoSchedule --overwrite
	kubectl taint nodes deschedbench node-role.kubernetes.io/master=:NoSchedule --overwrite

minikube-delete: ## Delete the dedicated minikube cluster
	minikube delete -p deschedbench

setup: ## Full local setup: minikube + monitoring + dashboards + descheduler ServiceMonitor
	$(MAKE) minikube-up
	$(MAKE) monitoring-up
	$(MAKE) dashboards-import
	$(MAKE) descheduler-servicemonitor

preflight: ## Verify the cluster is ready for a benchmark run
	@$(DESCHBENCH) preflight

bench-maintenance: ## Run maintenance scenario without descheduler (baseline)
	@$(DESCHBENCH) benchmark --pods $(PODS) --profile baseline --out results/baseline.json

bench-maintenance-descheduler: ## Run maintenance scenario with descheduler profile
	@$(DESCHBENCH) benchmark --pods $(PODS) --profile $(PROFILE) --out results/descheduler.json

descheduler-logs: ## Show logs for the latest descheduler job
	@set -euo pipefail; \
	entry=$$(kubectl get jobs -A --sort-by=.metadata.creationTimestamp \
	  -o jsonpath='{range .items[*]}{.metadata.namespace}{" "}{.metadata.name}{"\n"}{end}' \
	  | grep 'deschedbench-descheduler' | tail -n 1); \
	if [ -z "$$entry" ]; then echo "No deschedbench descheduler jobs found."; exit 1; fi; \
	ns=$$(echo "$$entry" | awk '{print $$1}'); \
	name=$$(echo "$$entry" | awk '{print $$2}'); \
	echo "Logs for $$ns/$$name"; \
	kubectl -n $$ns logs job/$$name --all-containers --tail=200

monitoring-up: ## Install kube-prometheus-stack for local monitoring
	helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
	helm repo update
	kubectl create namespace monitoring
	helm install prometheus prometheus-community/kube-prometheus-stack \
	  -n monitoring \
	  -f deploy/monitoring/kube-prometheus-stack-values-minikube.yaml

dashboards-import: ## Import Grafana dashboards as ConfigMap
	kubectl create configmap deschedbench-dashboards \
	  -n monitoring \
	  --from-file=deploy/dashboards/descheduler-impact.json \
	  --from-file=deploy/dashboards/apiserver-pressure.json \
	  --from-file=deploy/dashboards/run-timeline.json
	kubectl label configmap deschedbench-dashboards -n monitoring grafana_dashboard=1

descheduler-servicemonitor: ## Install ServiceMonitor for Descheduler metrics
	kubectl apply -f deploy/monitoring/servicemonitors/descheduler-servicemonitor.yaml

grafana-port-forward: ## Port-forward Grafana to localhost:3000
	kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80

prometheus-port-forward: ## Port-forward Prometheus to localhost:9090
	kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090

cleanup: ## Delete all deschedbench namespaces
	$(DESCHBENCH) cleanup

fmt: ## Format Go files
	go fmt ./...

tidy: ## Tidy go.mod/go.sum
	go mod tidy

format: ## Format and tidy modules
	$(MAKE) fmt
	$(MAKE) tidy

test: ## Run Go tests
	GOCACHE=/tmp/go-build GOSUMDB=off go test ./...

test-ci: ## Run Go tests with isolated module/cache dirs
	GOMODCACHE=/tmp/gomodcache GOPATH=/tmp/gopath GOCACHE=/tmp/go-build GOSUMDB=off go test ./...

help: ## Show available targets
	@awk 'BEGIN {FS=":.*## "}; /^[a-zA-Z0-9_.-]+:.*## / {printf "%-30s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
