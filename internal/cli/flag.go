package cli

import "github.com/spf13/cobra"

func flagToBoolPtr(cmd *cobra.Command, flagName string, value bool) *bool {
	if !cmd.Flags().Changed(flagName) {
		return nil
	}

	return &value
}

func flagToIntPtr(cmd *cobra.Command, flagName string, value int) *int {
	if !cmd.Flags().Changed(flagName) {
		return nil
	}

	return &value
}
