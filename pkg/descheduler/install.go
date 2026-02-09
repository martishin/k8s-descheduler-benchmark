package descheduler

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

type Config struct {
	Namespace    string
	Image        string
	PolicyYAML   string
	CronSchedule string
	JobName      string
}

func EnsureInstalled(ctx context.Context, client kubernetes.Interface, cfg Config) error {
	manifests, err := renderBaseManifests(cfg)
	if err != nil {
		return err
	}
	for _, manifest := range manifests {
		if err := applyManifest(ctx, client, manifest); err != nil {
			return err
		}
	}
	return nil
}

func RunOnce(ctx context.Context, client kubernetes.Interface, cfg Config) error {
	manifest, err := renderJobManifest(cfg)
	if err != nil {
		return err
	}
	return applyManifest(ctx, client, manifest)
}
