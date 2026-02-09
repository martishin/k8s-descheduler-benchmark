package benchmark

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"k8s-descheduler-benchmark/pkg/descheduler"
	"k8s-descheduler-benchmark/pkg/k8s"
	"k8s-descheduler-benchmark/pkg/logging"
	"k8s-descheduler-benchmark/pkg/report"
	"k8s-descheduler-benchmark/pkg/workloads"

	"k8s.io/apimachinery/pkg/util/wait"
)

func (m *maintenanceRunner) mark(name string, attrs ...slog.Attr) error {
	m.logger.Info(formatPhaseMessage(name), attrsToArgs(attrs)...)
	if m.cfg.RecordPhase != nil {
		if err := m.cfg.RecordPhase(name); err != nil {
			return err
		}
	}
	return nil
}

func (m *maintenanceRunner) prepareWorkloads() error {
	if err := k8s.EnsureNamespace(m.ctx, m.client, m.cfg.Namespace); err != nil {
		return err
	}
	if err := m.mark("workload:create",
		logging.StringField("workload", m.workloadName),
		logging.StringField("pods", fmt.Sprintf("%d", m.totalPods)),
	); err != nil {
		return err
	}
	if err := workloads.EnsureWorkloads(m.ctx, m.client, workloads.WorkloadConfig{
		Namespace:   m.cfg.Namespace,
		NamePrefix:  m.workloadName,
		Labels:      m.cfg.Labels,
		Mix:         m.cfg.WorkloadMix,
		SizeClasses: m.cfg.SizeClasses,
		PodImage:    m.cfg.WorkloadImage,
		PodLabels:   m.cfg.Labels,
	}); err != nil {
		return err
	}
	m.logger.Info("workload creation done",
		logging.StringField("workload", m.workloadName),
		logging.StringField("pods", fmt.Sprintf("%d", m.totalPods)),
	)
	m.logger.Info("workload waiting start",
		logging.StringField("workload", m.workloadName),
		logging.StringField("pods", fmt.Sprintf("%d", m.totalPods)),
	)
	if err := workloads.WaitForWorkloadsReady(m.ctx, m.client, m.cfg.Namespace, m.workloadName, m.cfg.WorkloadMix, m.cfg.WaitTimeout); err != nil {
		logSchedulingSummary(m.ctx, m.client, m.cfg.Namespace, m.cfg.LabelSelector, m.logger)
		return err
	}
	if err := m.mark("workload:ready",
		logging.StringField("workload", m.workloadName),
		logging.StringField("pods", fmt.Sprintf("%d", m.totalPods)),
	); err != nil {
		return err
	}
	return nil
}

func (m *maintenanceRunner) snapshotBefore() error {
	return m.mark("snapshot:before")
}

func (m *maintenanceRunner) validateIterations(iterations int) error {
	if iterations <= 1 {
		return nil
	}
	nodes, err := k8s.ListNodes(m.ctx, m.client, "")
	if err != nil {
		return err
	}
	workers := countSchedulableWorkers(nodes)
	if workers < 3 {
		return fmt.Errorf("need at least 3 worker nodes for %d maintenance iterations (found %d)", iterations, workers)
	}
	return nil
}

func (m *maintenanceRunner) selectDrainNode() error {
	nodes, err := k8s.ListNodes(m.ctx, m.client, "")
	if err != nil {
		return err
	}
	var drainNode string
	if len(m.drainedNodes) == 0 {
		drainNode, err = pickDrainNode(nodes)
	} else {
		drainNode, err = pickDrainNodeExcluding(nodes, m.drainedNodes)
	}
	if err != nil {
		return err
	}
	m.drainNode = drainNode
	m.drainedNodes[drainNode] = struct{}{}
	m.logger.Info("selected drain node",
		logging.StringField("node", drainNode),
		logging.StringField("iteration", fmt.Sprintf("%d", m.iteration)),
	)
	return nil
}

func (m *maintenanceRunner) cordonAndDrain() error {
	if err := m.mark("cordon:start",
		logging.StringField("node", m.drainNode),
		logging.StringField("iteration", fmt.Sprintf("%d", m.iteration)),
	); err != nil {
		return err
	}
	if err := k8s.CordonNode(m.ctx, m.client, m.drainNode); err != nil {
		return err
	}
	if err := m.mark("cordon:done",
		logging.StringField("node", m.drainNode),
		logging.StringField("iteration", fmt.Sprintf("%d", m.iteration)),
	); err != nil {
		return err
	}

	podsOnNode, err := countPodsOnNode(m.ctx, m.client, m.cfg.Namespace, m.cfg.LabelSelector, m.drainNode)
	if err == nil {
		if err := m.mark("drain:start",
			logging.StringField("node", m.drainNode),
			logging.StringField("pods", fmt.Sprintf("%d", podsOnNode)),
			logging.StringField("iteration", fmt.Sprintf("%d", m.iteration)),
		); err != nil {
			return err
		}
	} else {
		if err := m.mark("drain:start",
			logging.StringField("node", m.drainNode),
			logging.StringField("iteration", fmt.Sprintf("%d", m.iteration)),
		); err != nil {
			return err
		}
	}
	if err := k8s.DrainNode(m.ctx, m.client, m.drainNode, k8s.DrainOptions{Namespace: m.cfg.Namespace, LabelSelector: m.cfg.LabelSelector, Timeout: m.cfg.WaitTimeout}); err != nil {
		return err
	}
	if err := m.mark("drain:done",
		logging.StringField("node", m.drainNode),
		logging.StringField("iteration", fmt.Sprintf("%d", m.iteration)),
	); err != nil {
		return err
	}
	return nil
}

func (m *maintenanceRunner) waitForReschedule() error {
	expectedPods := m.totalPods
	if err := k8s.WaitForPodsReady(m.ctx, m.client, m.cfg.Namespace, m.cfg.LabelSelector, expectedPods, m.cfg.WaitTimeout); err != nil {
		if errors.Is(err, wait.ErrWaitTimeout) {
			if summary, sumErr := k8s.SummarizeScheduling(m.ctx, m.client, m.cfg.Namespace, m.cfg.LabelSelector); sumErr == nil {
				return fmt.Errorf("pods not ready after %s: ready %d/%d, pending %d, reasons: %s",
					m.cfg.WaitTimeout.String(),
					summary.Ready,
					expectedPods,
					summary.Pending,
					report.FormatScheduleMessages(summary.Messages),
				)
			}
		}
		logSchedulingSummary(m.ctx, m.client, m.cfg.Namespace, m.cfg.LabelSelector, m.logger)
		return err
	}
	m.logger.Info("pods ready after drain", logging.StringField("pods", fmt.Sprintf("%d", expectedPods)))
	return m.mark("reschedule:ready",
		logging.StringField("pods", fmt.Sprintf("%d", expectedPods)),
		logging.StringField("iteration", fmt.Sprintf("%d", m.iteration)),
	)
}

func (m *maintenanceRunner) uncordon() error {
	if err := m.mark("uncordon:start",
		logging.StringField("node", m.drainNode),
		logging.StringField("iteration", fmt.Sprintf("%d", m.iteration)),
	); err != nil {
		return err
	}
	if err := k8s.UncordonNode(m.ctx, m.client, m.drainNode); err != nil {
		return err
	}
	return m.mark("uncordon:done",
		logging.StringField("node", m.drainNode),
		logging.StringField("iteration", fmt.Sprintf("%d", m.iteration)),
	)
}

func (m *maintenanceRunner) capturePreEvictions() error {
	if pods, err := k8s.ListPods(m.ctx, m.client, m.cfg.Namespace, m.cfg.LabelSelector); err == nil {
		m.preEvictLabels = k8s.PodNameToAppLabel(pods)
	}
	return nil
}

func (m *maintenanceRunner) ensureDescheduler() error {
	if m.cfg.DeschedulerPolicy == "" {
		return nil
	}
	if err := m.mark("descheduler:install"); err != nil {
		return err
	}
	if err := descheduler.EnsureInstalled(m.ctx, m.client, descheduler.Config{
		Namespace:    m.cfg.DeschedulerNS,
		Image:        m.cfg.DeschedulerImage,
		PolicyYAML:   m.cfg.DeschedulerPolicy,
		CronSchedule: m.cfg.DeschedulerCron,
	}); err != nil {
		return err
	}
	m.logger.Info("descheduler installed")
	return nil
}

func (m *maintenanceRunner) runDescheduler() error {
	if m.cfg.DeschedulerPolicy == "" {
		return nil
	}
	if err := m.mark("descheduler:run", logging.StringField("iteration", fmt.Sprintf("%d", m.iteration))); err != nil {
		return err
	}
	jobName := fmt.Sprintf("deschedbench-descheduler-%s-%d", m.cfg.RunID, m.iteration)
	if err := descheduler.RunOnce(m.ctx, m.client, descheduler.Config{
		Namespace:    m.cfg.DeschedulerNS,
		Image:        m.cfg.DeschedulerImage,
		PolicyYAML:   m.cfg.DeschedulerPolicy,
		CronSchedule: m.cfg.DeschedulerCron,
		JobName:      jobName,
	}); err != nil {
		return err
	}
	m.logger.Info("descheduler job created",
		logging.StringField("job", jobName),
		logging.StringField("iteration", fmt.Sprintf("%d", m.iteration)),
	)
	return m.mark("descheduler:done", logging.StringField("iteration", fmt.Sprintf("%d", m.iteration)))
}

func (m *maintenanceRunner) waitPostUncordon() error {
	if m.cfg.PostUncordonWait <= 0 {
		return nil
	}
	m.logger.Info("waiting after uncordon",
		logging.StringField("duration", m.cfg.PostUncordonWait.String()),
		logging.StringField("iteration", fmt.Sprintf("%d", m.iteration)),
	)
	time.Sleep(m.cfg.PostUncordonWait)
	return nil
}

func (m *maintenanceRunner) snapshotAfter() error {
	return m.mark("snapshot:after")
}

func (m *maintenanceRunner) collectEvictions() []k8s.EvictionRecord {
	var evictions []k8s.EvictionRecord
	postPods, err := k8s.ListPods(m.ctx, m.client, m.cfg.Namespace, m.cfg.LabelSelector)
	if err == nil {
		evictions, _ = k8s.CollectEvictions(m.ctx, m.client, m.cfg.Namespace, m.preEvictLabels, postPods)
	}
	return evictions
}
