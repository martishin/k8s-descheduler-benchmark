# Descheduler Performance & Impact Benchmark (`deschedbench`)

A benchmarking tool to evaluate **Kubernetes Descheduler** effectiveness and side effects during node maintenance
(cordon/drain/uncordon) by measuring balance, scheduling outcomes, and operational cost.

## Overview

`deschedbench`:
- creates lightweight Deployment-based workloads in an isolated namespace
- runs **two** maintenance iterations (cordon → drain → uncordon)
- optionally runs Descheduler as a one-shot Job **after each uncordon**
- samples balance metrics over time and writes a single JSON result file
- exposes Prometheus metrics for Grafana dashboards

Balance metrics are computed **only for the benchmark namespace** to isolate descheduler impact.

Pinned versions:
- **Kubernetes**: `v1.32.0` (minikube)
- **Descheduler**: `registry.k8s.io/descheduler/descheduler:v0.32.2`

## Prerequisites

- **Go 1.24+**
- **Minikube** (recommended for local runs)
- **kubectl** configured to the `deschedbench` minikube profile
- **Helm** (for monitoring stack)

## 1) Minikube setup

The benchmark performs two maintenance iterations and expects **3 worker nodes** (4 nodes total: 1 control-plane + 3 workers).

```bash
minikube start -p deschedbench --kubernetes-version=v1.32.0 --cpus=2 --memory=4096 --nodes=4 \
  --extra-config=kubelet.max-pods=250 \
  --extra-config=controller-manager.bind-address=0.0.0.0 \
  --extra-config=scheduler.bind-address=0.0.0.0 \
  --extra-config=etcd.listen-metrics-urls=http://0.0.0.0:2381
kubectl config use-context deschedbench
kubectl taint nodes deschedbench node-role.kubernetes.io/control-plane=:NoSchedule --overwrite
kubectl taint nodes deschedbench node-role.kubernetes.io/master=:NoSchedule --overwrite
```

Makefile shortcut:

```bash
make minikube-up
```

`minikube-up` taints the control-plane node so workloads land on workers.

To delete the dedicated cluster:

```bash
make minikube-delete
```

Full setup shortcut (minikube + monitoring + dashboards + ServiceMonitor):

```bash
make setup
```

## 2) Monitoring setup

We use `kube-prometheus-stack` to scrape control-plane metrics and `deschedbench` metrics (port 8080 by default).

### Install Prometheus stack

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

kubectl create namespace monitoring
helm install prometheus prometheus-community/kube-prometheus-stack \
  -n monitoring \
  -f deploy/monitoring/kube-prometheus-stack-values-minikube.yaml
```

The values file scrapes `host.docker.internal:8080` (macOS + Docker driver). If you are on Linux or another driver,
replace the target with `host.minikube.internal:8080` or the value of `minikube ip` with an exposed port.

### Import dashboards

```bash
kubectl create configmap deschedbench-dashboards \
  -n monitoring \
  --from-file=deploy/dashboards/descheduler-impact.json \
  --from-file=deploy/dashboards/apiserver-pressure.json \
  --from-file=deploy/dashboards/run-timeline.json

kubectl label configmap deschedbench-dashboards -n monitoring grafana_dashboard=1
```

### Descheduler ServiceMonitor

The tool creates a metrics Service in the Descheduler namespace. Apply the ServiceMonitor once (it watches all namespaces):

```bash
kubectl apply -f deploy/monitoring/servicemonitors/descheduler-servicemonitor.yaml
```

If you want to restrict which namespaces are scraped, edit the `namespaceSelector` in the ServiceMonitor YAML.

Makefile shortcut:

```bash
make dashboards-import
make descheduler-servicemonitor
```

### Grafana access

```bash
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80
# http://localhost:3000 (admin / admin)
```

Optional Prometheus port-forward:

```bash
kubectl port-forward -n monitoring svc/prometheus-kube-prometheus-prometheus 9090:9090
```

## 3) Tool usage

### Benchmark

```bash
# Baseline (no descheduler)
go run ./cmd/deschedbench benchmark --pods 60 --profile baseline

# Descheduler enabled (runs once after each uncordon)
go run ./cmd/deschedbench benchmark --pods 60 --profile low-node-utilization

# Override pod requests
go run ./cmd/deschedbench benchmark --pods 60 --cpu 200m --mem 256Mi --profile low-node-utilization

# Write results to a custom file
go run ./cmd/deschedbench benchmark --pods 60 --profile baseline --out results/custom.json
```

<details>
<summary>Example command output (trimmed)</summary>

```text
monkey@local:/Volumes/SSD/development/code/k8s-descheduler-benchmark$ make bench-maintenance-descheduler PROFILE=low-node-utilization
time=2026-02-09T02:50:12 level=INFO msg="benchmark namespace" value=deschedbench-20260209-025012
time=2026-02-09T02:50:12 level=INFO msg="results file path" value=results/descheduler.json
time=2026-02-09T02:50:12 level=INFO msg="metrics server start" port=8080
time=2026-02-09T02:50:12 level=INFO msg="run id" value=20260209-025012
time=2026-02-09T02:50:12 level=INFO msg="starting maintenance scenario"
time=2026-02-09T02:50:12 level=INFO msg="namespace created" name=deschedbench-20260209-025012
time=2026-02-09T02:50:12 level=INFO msg="workload creation start" workload=deschedbench pods=60
time=2026-02-09T02:50:12 level=INFO msg="workload creation done" workload=deschedbench pods=60
time=2026-02-09T02:50:12 level=INFO msg="workload waiting start" workload=deschedbench pods=60
time=2026-02-09T02:50:18 level=INFO msg="workload waiting done" workload=deschedbench pods=60
time=2026-02-09T02:50:18 level=INFO msg="snapshot before start"
time=2026-02-09T02:50:18 level=INFO msg="snapshot before done" metric="pods per node (count)" pods_per_node="deschedbench=0 deschedbench-m02=20 deschedbench-m03=20 deschedbench-m04=20"
time=2026-02-09T02:50:18 level=INFO msg="descheduler install"
time=2026-02-09T02:50:18 level=INFO msg="descheduler installed"
time=2026-02-09T02:50:18 level=INFO msg="maintenance iteration" iteration=1
time=2026-02-09T02:50:18 level=INFO msg="selected drain node" node=deschedbench-m02 iteration=1
time=2026-02-09T02:50:18 level=INFO msg="cordon start" node=deschedbench-m02 iteration=1
time=2026-02-09T02:50:18 level=INFO msg="cordon done" node=deschedbench-m02 iteration=1
time=2026-02-09T02:50:19 level=INFO msg="drain start" node=deschedbench-m02 pods=20 iteration=1
time=2026-02-09T02:50:23 level=INFO msg="drain done" node=deschedbench-m02 iteration=1
time=2026-02-09T02:50:23 level=INFO msg="pods ready after drain" pods=60
time=2026-02-09T02:50:23 level=INFO msg="reschedule ready" pods=60 iteration=1
time=2026-02-09T02:50:23 level=INFO msg="uncordon start" node=deschedbench-m02 iteration=1
time=2026-02-09T02:50:23 level=INFO msg="uncordon done" node=deschedbench-m02 iteration=1
time=2026-02-09T02:50:23 level=INFO msg="descheduler run" iteration=1
time=2026-02-09T02:50:23 level=INFO msg="descheduler job created" job=deschedbench-descheduler-20260209-025012-1 iteration=1
time=2026-02-09T02:50:23 level=INFO msg="descheduler done" iteration=1
time=2026-02-09T02:50:23 level=INFO msg="waiting after uncordon" duration=1m0s iteration=1
time=2026-02-09T02:51:23 level=INFO msg="maintenance iteration" iteration=2
time=2026-02-09T02:51:23 level=INFO msg="selected drain node" node=deschedbench-m03 iteration=2
time=2026-02-09T02:51:23 level=INFO msg="cordon start" node=deschedbench-m03 iteration=2
time=2026-02-09T02:51:23 level=INFO msg="cordon done" node=deschedbench-m03 iteration=2
time=2026-02-09T02:51:23 level=INFO msg="drain start" node=deschedbench-m03 pods=23 iteration=2
time=2026-02-09T02:51:28 level=INFO msg="drain done" node=deschedbench-m03 iteration=2
time=2026-02-09T02:51:28 level=INFO msg="pods ready after drain" pods=60
time=2026-02-09T02:51:28 level=INFO msg="reschedule ready" pods=60 iteration=2
time=2026-02-09T02:51:28 level=INFO msg="uncordon start" node=deschedbench-m03 iteration=2
time=2026-02-09T02:51:28 level=INFO msg="uncordon done" node=deschedbench-m03 iteration=2
time=2026-02-09T02:51:28 level=INFO msg="descheduler run" iteration=2
time=2026-02-09T02:51:28 level=INFO msg="descheduler job created" job=deschedbench-descheduler-20260209-025012-2 iteration=2
time=2026-02-09T02:51:28 level=INFO msg="descheduler done" iteration=2
time=2026-02-09T02:51:28 level=INFO msg="waiting after uncordon" duration=1m0s iteration=2
time=2026-02-09T02:52:28 level=INFO msg="snapshot after benchmark start"
time=2026-02-09T02:52:28 level=INFO msg="snapshot after benchmark done" metric="pods per node (count)" pods_per_node="deschedbench=0 deschedbench-m02=23 deschedbench-m03=14 deschedbench-m04=23"
time=2026-02-09T02:52:28 level=INFO msg="run summary" duration=135.5s rebalance_time=-1.0s before_pods_stddev=8.660 after_pods_stddev=9.407 before_pods="deschedbench=0 deschedbench-m02=20 deschedbench-m03=20 deschedbench-m04=20" after_pods="deschedbench=0 deschedbench-m02=23 deschedbench-m03=14 deschedbench-m04=23"
time=2026-02-09T02:52:28 level=INFO msg="benchmark completed"
time=2026-02-09T02:52:28 level=INFO msg="namespace cleanup start" reason=success namespace=deschedbench-20260209-025012
time=2026-02-09T02:52:41 level=INFO msg="namespace cleanup done" namespace=deschedbench-20260209-025012
time=2026-02-09T02:52:41 level=INFO msg="results output" path=results/descheduler.json
```
</details>

### Preflight

```bash
go run ./cmd/deschedbench preflight
```

Fails if any nodes are already cordoned/unschedulable.

### Cleanup

```bash
go run ./cmd/deschedbench cleanup
```

Note: the benchmark command always runs cleanup (success, failure, or Ctrl+C): it deletes the current
`deschedbench-<timestamp>` namespace and uncordons any unschedulable nodes. `make cleanup` removes all
`deschedbench-*` namespaces.

### Descheduler logs (latest job)

```bash
make descheduler-logs
```

## 4) Results

Each run writes a single JSON document to `results/baseline.json` (baseline profile) or
`results/descheduler.json` (any non-baseline profile) by default. The file includes:

- config
- phases
- summary
- samples
- before/after snapshots
- evictions

Use `--out` to write to a custom path. The results are stored under `results/`.

### Interpreting results

Run **baseline** and **descheduler** with the same inputs, then compare:

1) **Balance improvement (primary signal)**
   - Look at `summary.before.pods_stddev` vs `summary.after.pods_stddev`
   - Lower **after** stddev with descheduler vs baseline = better balance

2) **Node distribution**
   - Check `before_snapshot.nodes` and `after_snapshot.nodes`
   - You want **pods per node** to converge toward even distribution after maintenance

3) **Evictions and safety**
   - `evictions` should be present for descheduler runs
   - If evictions are excessive or do not improve balance, adjust thresholds in the policy

4) **Unschedulable pods**
   - `summary.before/after.unschedulable_pods` should remain 0

Quick workflow:
```bash
make bench-maintenance
make bench-maintenance-descheduler PROFILE=low-node-utilization
```

Then compare:
- `results/baseline.json` vs `results/descheduler.json`
- Focus on **pods_stddev** and **pods per node** after the final snapshot

If you see **no improvement**:
- Adjust thresholds in `deploy/descheduler/policies/low-node-utilization.yaml`
- Increase eviction limits at the top of the policy file

Example excerpt (trimmed):

Baseline:
```json
{
  "summary": {
    "before": { "pods_stddev": 8.660 },
    "after":  { "pods_stddev": 15.000 }
  },
  "after_snapshot": {
    "nodes": {
      "deschedbench-m02": { "pods": 30 },
      "deschedbench-m03": { "pods": 0 },
      "deschedbench-m04": { "pods": 30 }
    }
  }
}
```

LowNodeUtilization + RemoveDuplicates:
```json
{
  "summary": {
    "before": { "pods_stddev": 8.660 },
    "after":  { "pods_stddev": 2.000 }
  },
  "after_snapshot": {
    "nodes": {
      "deschedbench-m02": { "pods": 23 },
      "deschedbench-m03": { "pods": 14 },
      "deschedbench-m04": { "pods": 23 }
    }
  }
}
```

## 5) Makefile shortcuts

```bash
make minikube-up
make setup
make monitoring-up
make dashboards-import
make descheduler-servicemonitor
make preflight
make bench-maintenance
make bench-maintenance-descheduler PROFILE=low-node-utilization
make descheduler-logs
make grafana-port-forward
make prometheus-port-forward
make cleanup
make fmt
make tidy
make format
make test
```

## 6) Safety guards

- The tool **only runs on the `deschedbench` context**.
- All workloads live in `deschedbench-<timestamp>` namespaces and **only those namespaces are touched**.
- Descheduler evictions are scoped by label (`deschedbench=true`) to keep changes within the benchmark workload.

## Notes on Descheduler policy

`deschedbench` ships **v1alpha2** policies under `deploy/descheduler/policies/`. The CLI selects a default policy
based on `--profile`. To customize, edit the policy file for that profile.
Descheduler Kubernetes resources are templated from YAMLs under `deploy/descheduler/manifests/`.

## Metrics endpoint

The tool exposes Prometheus metrics at:

```bash
curl http://localhost:8080/metrics
```

Adjust the port using `--metrics-port` if needed.
