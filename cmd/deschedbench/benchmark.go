package main

import (
	"context"

	"k8s-descheduler-benchmark/pkg/k8s"
	"k8s-descheduler-benchmark/pkg/logging"
	benchsvc "k8s-descheduler-benchmark/pkg/service/benchmark"

	"github.com/spf13/cobra"
)

var (
	podsTotal  int32
	podCPU     string
	podMem     string
	profile    string
	outputPath string
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run a single benchmark scenario",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, info, err := k8s.NewClient(clientQPS, clientBurst)
		if err != nil {
			return err
		}

		runner := benchsvc.Runner{
			Client:      client,
			Logger:      logging.GetLogger(),
			MetricsPort: metricsPort,
		}
		return runner.Run(context.Background(), benchsvc.RunConfig{
			PodsTotal:  podsTotal,
			PodCPU:     podCPU,
			PodMemory:  podMem,
			Profile:    profile,
			OutputPath: outputPath,
			Context:    info.Context,
			Server:     info.Server,
		})
	},
}

func init() {
	benchmarkCmd.Flags().Int32Var(&podsTotal, "pods", 60, "Number of pods to schedule")
	benchmarkCmd.Flags().StringVar(&podCPU, "cpu", "100m", "CPU request per pod")
	benchmarkCmd.Flags().StringVar(&podMem, "mem", "128Mi", "Memory request per pod")
	benchmarkCmd.Flags().StringVar(&profile, "profile", "baseline", "Descheduler profile (baseline, low-node-utilization, low-node-utilization+duplicates, taints, topology-spread)")
	benchmarkCmd.Flags().StringVar(&outputPath, "out", "", "Write JSON output to a file (default: results/baseline.json or results/descheduler.json)")

	rootCmd.AddCommand(benchmarkCmd)
}
