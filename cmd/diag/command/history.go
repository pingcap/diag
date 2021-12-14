package command

import (
	"github.com/pingcap/diag/pkg/packager"
	"github.com/spf13/cobra"
)

func newHistoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "show upload history",
		RunE: func(cmd *cobra.Command, args []string) error {
			his, err := packager.LoadHistroy()
			if err != nil {
				return err
			}
			his.PrintList()
			return nil
		},
	}

	return cmd
}
