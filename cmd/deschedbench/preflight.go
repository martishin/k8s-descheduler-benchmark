package main

import (
	"context"

	"k8s-descheduler-benchmark/pkg/k8s"
	"k8s-descheduler-benchmark/pkg/logging"
	"k8s-descheduler-benchmark/pkg/service/cleanup"

	"github.com/spf13/cobra"
)

var preflightCmd = &cobra.Command{
	Use:   "preflight",
	Short: "Verify the cluster is ready for a benchmark run",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := k8s.NewClient(clientQPS, clientBurst)
		if err != nil {
			return err
		}

		svc := cleanup.NewCleanupService(client, logging.GetLogger())
		if err := svc.Preflight(context.Background()); err != nil {
			return err
		}
		logging.GetLogger().Info("preflight ok")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(preflightCmd)
}
