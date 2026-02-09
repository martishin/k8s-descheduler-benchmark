package report

import (
	"fmt"
	"sort"
	"strings"

	"k8s-descheduler-benchmark/pkg/metrics"
)

func FormatNodePods(snapshot metrics.Snapshot) string {
	if len(snapshot.Nodes) == 0 {
		return "-"
	}
	names := make([]string, 0, len(snapshot.Nodes))
	for name := range snapshot.Nodes {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, fmt.Sprintf("%s=%d", name, snapshot.Nodes[name].Pods))
	}
	return strings.Join(parts, " ")
}

func FormatScheduleMessages(messages map[string]int) string {
	if len(messages) == 0 {
		return "none"
	}
	type entry struct {
		msg   string
		count int
	}
	entries := make([]entry, 0, len(messages))
	for msg, count := range messages {
		entries = append(entries, entry{msg: msg, count: count})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count == entries[j].count {
			return entries[i].msg < entries[j].msg
		}
		return entries[i].count > entries[j].count
	})
	limit := 3
	if len(entries) < limit {
		limit = len(entries)
	}
	parts := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		parts = append(parts, fmt.Sprintf("%s (x%d)", entries[i].msg, entries[i].count))
	}
	return strings.Join(parts, "; ")
}
