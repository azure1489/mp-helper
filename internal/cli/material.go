package cli

import "github.com/spf13/cobra"

func newMaterialCmd() *cobra.Command {
	m := &cobra.Command{Use: "material", Short: "素材操作"}

	var mtype string
	up := &cobra.Command{
		Use:   "upload <file>",
		Short: "上传永久图片素材，返回 media_id 与 url",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			res, err := c.UploadMaterial(args[0], mtype)
			if err != nil {
				return err
			}
			return printJSON(cmd, res)
		},
	}
	up.Flags().StringVar(&mtype, "type", "image", "素材类型（image|thumb）")
	m.AddCommand(up)
	return m
}
