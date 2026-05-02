package cmd

import (
	"fmt"

	"github.com/pc0stas/tango/internal/config"
	"github.com/spf13/cobra"
)

func NewValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <workflow.yaml>",
		Short: "Validate workflow syntax",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.ParseWorkflow(args[0])
			if err != nil {
				fmt.Printf("✗ Validation failed: %v\n", err)
				return err
			}
			fmt.Println("✓ Workflow is valid")
			return nil
		},
	}
}
