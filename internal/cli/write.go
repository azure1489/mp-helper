package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/azure1489/mp-helper/internal/types"
	"github.com/spf13/cobra"
)

func newBodyImageCmd() *cobra.Command {
	root := &cobra.Command{Use: "body-image", Short: "正文图片（非永久素材）"}
	root.AddCommand(&cobra.Command{
		Use: "upload <file>", Args: cobra.ExactArgs(1), Short: "原字节上传 jpg/png（小于 1 MB），返回 url",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			out, err := c.UploadBodyImage(args[0])
			if err != nil {
				return err
			}
			return printJSON(cmd, out)
		},
	})
	return root
}

func newDraftUpdateCmd() *cobra.Command {
	var mediaID, articleFile string
	var index int
	cmd := &cobra.Command{
		Use: "update", Args: cobra.NoArgs, Short: "更新指定草稿 ID/index；完整 article JSON，不创建或自动重试",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("index") {
				return fmt.Errorf("--index must be explicitly provided (0 is valid)")
			}
			if mediaID == "" || articleFile == "" {
				return fmt.Errorf("--media-id and --article-file are required")
			}
			data, err := os.ReadFile(articleFile)
			if err != nil {
				return err
			}
			req := types.DraftUpdateRequest{MediaID: mediaID, Index: &index, Articles: json.RawMessage(data)}
			if _, err := req.Validate(); err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			out, err := c.UpdateDraft(req)
			if err != nil {
				return err
			}
			return printJSON(cmd, out)
		},
	}
	cmd.Flags().StringVar(&mediaID, "media-id", "", "现有草稿 media_id（必填）")
	cmd.Flags().IntVar(&index, "index", 0, "文章位置（必须显式指定，0 为首篇）")
	cmd.Flags().StringVar(&articleFile, "article-file", "", "含所有 Article 字段的 JSON 文件（必填）")
	return cmd
}
