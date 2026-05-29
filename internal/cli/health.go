package cli

import "github.com/spf13/cobra"

func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "检查 mp-server 是否可用",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			res, err := c.Health()
			if err != nil {
				return err
			}
			return printJSON(cmd, res)
		},
	}
}
