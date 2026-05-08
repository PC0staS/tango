package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pc0stas/tango/cmd"
	"github.com/pc0stas/tango/internal/config"
	"github.com/pc0stas/tango/internal/executor"
	"github.com/pc0stas/tango/internal/output"
	"github.com/spf13/cobra"
)

var Version = "1.0.10"

func main() {
	var dump bool

	rootCmd := &cobra.Command{
		Use:     "tango",
		Short:   "Distributed testing CLI",
		Version: Version,
	}

	testCmd := &cobra.Command{
		Use:   "test <workflow.yaml>",
		Short: "Execute a test workflow",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTest(args[0], dump)
		},
	}

	testCmd.Flags().BoolVar(&dump, "dump", false, "Dump full request and response details")

	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(cmd.NewValidateCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runTest(workflowFile string, dump bool) error {
	// 1. Parse
	workflow, err := config.ParseWorkflow(workflowFile)
	if err != nil {
		return fmt.Errorf("parse failed: %w", err)
	}

	// 2. Execute
	exec := executor.NewExecutor(workflow)
	exec.Dump = dump
	result, err := exec.Run(context.Background())
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	// 3. Output
	fmt.Print(output.FormatText(result))

	// 4. Exit code
	if !result.Success {
		os.Exit(1)
	}

	return nil
}
