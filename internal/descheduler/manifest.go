package descheduler

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var baseManifestFiles = []string{
	"deploy/descheduler/manifests/serviceaccount.yaml",
	"deploy/descheduler/manifests/rbac.yaml",
	"deploy/descheduler/manifests/policy-configmap.yaml",
	"deploy/descheduler/manifests/metrics-service.yaml",
}

const jobManifestFile = "deploy/descheduler/manifests/job.yaml"

func renderBaseManifests(cfg Config) ([]string, error) {
	if cfg.Namespace == "" {
		return nil, fmt.Errorf("descheduler namespace is required")
	}
	if cfg.PolicyYAML == "" {
		return nil, fmt.Errorf("policy YAML is required")
	}
	schedule := cfg.CronSchedule
	if schedule == "" {
		schedule = "*/1 * * * *"
	}
	return renderManifestSet(cfg, baseManifestFiles, schedule)
}

func indentPolicy(policy string, spaces int) string {
	policy = strings.TrimRight(policy, "\n")
	if policy == "" {
		return ""
	}
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(policy, "\n")
	for i, line := range lines {
		lines[i] = pad + line
	}
	return strings.Join(lines, "\n")
}

func renderJobManifest(cfg Config) (string, error) {
	if cfg.Namespace == "" {
		return "", fmt.Errorf("descheduler namespace is required")
	}
	if cfg.PolicyYAML == "" {
		return "", fmt.Errorf("policy YAML is required")
	}
	schedule := cfg.CronSchedule
	if schedule == "" {
		schedule = "*/1 * * * *"
	}
	manifests, err := renderManifestSet(cfg, []string{jobManifestFile}, schedule)
	if err != nil {
		return "", err
	}
	return manifests[0], nil
}

func renderManifestSet(cfg Config, paths []string, schedule string) ([]string, error) {
	jobName := cfg.JobName
	if jobName == "" {
		jobName = "deschedbench-descheduler"
	}
	manifests := make([]string, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(resolveManifestPath(path))
		if err != nil {
			return nil, err
		}
		manifest := string(data)
		manifest = strings.ReplaceAll(manifest, "{{NAMESPACE}}", cfg.Namespace)
		manifest = strings.ReplaceAll(manifest, "{{IMAGE}}", cfg.Image)
		manifest = strings.ReplaceAll(manifest, "{{SCHEDULE}}", schedule)
		manifest = strings.ReplaceAll(manifest, "{{JOB_NAME}}", jobName)
		manifest = strings.ReplaceAll(manifest, "{{POLICY_YAML}}", indentPolicy(cfg.PolicyYAML, 4))
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}

func resolveManifestPath(path string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return path
	}
	root := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
	return filepath.Join(root, path)
}
