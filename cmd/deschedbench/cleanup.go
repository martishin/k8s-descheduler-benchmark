package main

import (
	"context"
	"fmt"
	"time"

	"k8s-descheduler-benchmark/pkg/k8s"
	"k8s-descheduler-benchmark/pkg/logging"
	"k8s-descheduler-benchmark/pkg/service/cleanup"

	"github.com/spf13/cobra"
)

var (
	cleanupForce bool
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Delete namespaces created by deschedbench",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("cleanup does not accept positional arguments")
		}

		client, _, err := k8s.NewClient(clientQPS, clientBurst)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		cleanupSvc := cleanup.NewCleanupService(client, logging.GetLogger())
		return cleanupSvc.Run(ctx, cleanup.Scope{
			NamespacePrefix: "deschedbench-",
			Wait:            !cleanupForce,
		})
	},
}

func init() {
	cleanupCmd.Flags().BoolVar(&cleanupForce, "force", false, "Skip waiting for namespace deletion")
	rootCmd.AddCommand(cleanupCmd)
}
