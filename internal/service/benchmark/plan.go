package benchmark

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"k8s-descheduler-benchmark/internal/descheduler"
	"k8s-descheduler-benchmark/internal/workloads"
)

type Plan struct {
	RunID         string
	Namespace     string
	OutputPath    string
	Labels        map[string]string
	LabelSelector string
	Mix           workloads.Mix
	SizeClasses   map[string]workloads.SizeClass
	PolicyYAML    string
}

type PlanBuilder struct {
	Now func() time.Time
}

func NewPlanBuilder() *PlanBuilder {
	return &PlanBuilder{Now: time.Now}
}

func (b *PlanBuilder) Build(cfg RunConfig) (Plan, error) {
	if cfg.PodsTotal <= 0 {
		return Plan{}, fmt.Errorf("--pods must be > 0")
	}
	now := b.Now
	if now == nil {
		now = time.Now
	}

	runID := now().Format("20060102-150405")
	namespace := fmt.Sprintf("deschedbench-%s", runID)

	outPath := cfg.OutputPath
	if outPath == "" {
		outPath = defaultOutputPath(cfg.Profile)
	}

	mix := workloads.Mix{"small": cfg.PodsTotal}
	sizeClasses := map[string]workloads.SizeClass{
		"small": {Name: "small", CPU: cfg.PodCPU, Memory: cfg.PodMemory},
	}

	labels := map[string]string{
		"deschedbench":     "true",
		"deschedbench-run": runID,
	}

	policyYAML := ""
	if cfg.Profile != descheduler.ProfileBaseline {
		var err error
		policyYAML, err = loadPolicy(cfg.Profile, namespace)
		if err != nil {
			return Plan{}, err
		}
	}

	return Plan{
		RunID:         runID,
		Namespace:     namespace,
		OutputPath:    outPath,
		Labels:        labels,
		LabelSelector: labelsToSelector(labels),
		Mix:           mix,
		SizeClasses:   sizeClasses,
		PolicyYAML:    policyYAML,
	}, nil
}

func labelsToSelector(labels map[string]string) string {
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ",")
}

func defaultOutputPath(profile string) string {
	switch profile {
	case descheduler.ProfileBaseline:
		return filepath.Join(defaultResultsDir, "baseline.json")
	default:
		return filepath.Join(defaultResultsDir, "descheduler.json")
	}
}

func loadPolicy(profile, namespace string) (string, error) {
	path, err := defaultPolicyPath(profile)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(data), "{{NAMESPACE}}", namespace), nil
}

func defaultPolicyPath(profile string) (string, error) {
	switch profile {
	case descheduler.ProfileLowNodeUtilization:
		return "deploy/descheduler/policies/low-node-utilization.yaml", nil
	case descheduler.ProfileLowNodeUtilizationDupe:
		return "deploy/descheduler/policies/low-node-utilization+duplicates.yaml", nil
	case descheduler.ProfileTaints:
		return "deploy/descheduler/policies/taints.yaml", nil
	case descheduler.ProfileTopologySpread:
		return "deploy/descheduler/policies/topology-spread.yaml", nil
	case descheduler.ProfileBaseline:
		return "", fmt.Errorf("baseline does not use a policy")
	default:
		return "", fmt.Errorf("unknown profile %q", profile)
	}
}
