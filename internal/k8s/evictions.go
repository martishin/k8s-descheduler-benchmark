package k8s

import (
	"context"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type EvictionRecord struct {
	PodName           string    `json:"pod_name"`
	AppLabel          string    `json:"app_label"`
	NodeName          string    `json:"node_name"`
	Reason            string    `json:"reason"`
	Message           string    `json:"message"`
	EvictedAt         time.Time `json:"evicted_at"`
	RescheduledAt     time.Time `json:"rescheduled_at,omitempty"`
	RescheduleSeconds float64   `json:"reschedule_seconds"`
}

func CollectEvictions(ctx context.Context, client kubernetes.Interface, namespace string, prePodLabels map[string]string, postPods []corev1.Pod) ([]EvictionRecord, error) {
	events, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.kind=Pod",
	})
	if err != nil {
		return nil, err
	}

	readyTimes := readyTimesByAppLabel(postPods)

	records := make([]EvictionRecord, 0)
	for _, event := range events.Items {
		if event.Reason != "Evicted" {
			continue
		}
		podName := event.InvolvedObject.Name
		appLabel := prePodLabels[podName]
		if appLabel == "" {
			continue
		}
		evictedAt := eventTimestamp(&event)
		rec := EvictionRecord{
			PodName:   podName,
			AppLabel:  appLabel,
			NodeName:  event.Source.Host,
			Reason:    event.Reason,
			Message:   event.Message,
			EvictedAt: evictedAt,
		}
		if reschedAt, ok := findRescheduleTime(readyTimes[appLabel], evictedAt); ok {
			rec.RescheduledAt = reschedAt
			rec.RescheduleSeconds = reschedAt.Sub(evictedAt).Seconds()
		} else {
			rec.RescheduleSeconds = -1
		}
		records = append(records, rec)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].EvictedAt.Before(records[j].EvictedAt)
	})

	return records, nil
}

func PodNameToAppLabel(pods []corev1.Pod) map[string]string {
	out := make(map[string]string, len(pods))
	for _, pod := range pods {
		if appLabel := pod.Labels["app.kubernetes.io/name"]; appLabel != "" {
			out[pod.Name] = appLabel
		}
	}
	return out
}

func readyTimesByAppLabel(pods []corev1.Pod) map[string][]time.Time {
	out := map[string][]time.Time{}
	for _, pod := range pods {
		appLabel := pod.Labels["app.kubernetes.io/name"]
		if appLabel == "" {
			continue
		}
		readyAt := podReadyTime(&pod)
		if readyAt.IsZero() {
			continue
		}
		out[appLabel] = append(out[appLabel], readyAt)
	}
	for label := range out {
		sort.Slice(out[label], func(i, j int) bool {
			return out[label][i].Before(out[label][j])
		})
	}
	return out
}

func findRescheduleTime(times []time.Time, evictedAt time.Time) (time.Time, bool) {
	for _, t := range times {
		if !t.Before(evictedAt) {
			return t, true
		}
	}
	return time.Time{}, false
}

func podReadyTime(pod *corev1.Pod) time.Time {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
			return cond.LastTransitionTime.Time
		}
	}
	return time.Time{}
}

func eventTimestamp(event *corev1.Event) time.Time {
	if !event.EventTime.IsZero() {
		return event.EventTime.Time
	}
	if !event.LastTimestamp.IsZero() {
		return event.LastTimestamp.Time
	}
	if !event.FirstTimestamp.IsZero() {
		return event.FirstTimestamp.Time
	}
	return time.Now()
}
