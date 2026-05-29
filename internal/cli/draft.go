package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/azure1489/mp-helper/internal/types"
)

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func newDraftCmd() *cobra.Command {
	d := &cobra.Command{Use: "draft", Short: "草稿操作"}

	var title, contentFile, cover, author, digest, sourceURL string
	var showCover, needComment, fansComment bool

	create := &cobra.Command{
		Use:   "create",
		Short: "创建一篇图文草稿",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			if title == "" || contentFile == "" || cover == "" {
				return fmt.Errorf("--title, --content-file and --cover are required")
			}
			contentBytes, err := os.ReadFile(contentFile)
			if err != nil {
				return fmt.Errorf("read content file: %w", err)
			}

			// --cover 既可是已有 media_id，也可是本地图片路径（路径则先上传）。
			thumbID := cover
			if fileExists(cover) {
				mr, err := c.UploadMaterial(cover, "image")
				if err != nil {
					return fmt.Errorf("upload cover: %w", err)
				}
				thumbID = mr.MediaID
			}

			art := types.Article{
				Title:            title,
				Author:           author,
				Digest:           digest,
				Content:          string(contentBytes),
				ContentSourceURL: sourceURL,
				ThumbMediaID:     thumbID,
			}
			if showCover {
				art.ShowCoverPic = 1
			}
			if needComment {
				art.NeedOpenComment = 1
			}
			if fansComment {
				art.OnlyFansCanComment = 1
			}

			res, err := c.CreateDraft(types.DraftRequest{Articles: []types.Article{art}})
			if err != nil {
				return err
			}
			return printJSON(cmd, res)
		},
	}

	f := create.Flags()
	f.StringVar(&title, "title", "", "文章标题（必填）")
	f.StringVar(&contentFile, "content-file", "", "正文 HTML 文件路径（必填）")
	f.StringVar(&cover, "cover", "", "封面 thumb_media_id 或本地图片路径（必填）")
	f.StringVar(&author, "author", "", "作者")
	f.StringVar(&digest, "digest", "", "摘要")
	f.StringVar(&sourceURL, "source-url", "", "原文链接")
	f.BoolVar(&showCover, "show-cover-pic", false, "正文显示封面图")
	f.BoolVar(&needComment, "need-open-comment", false, "开启评论")
	f.BoolVar(&fansComment, "only-fans-can-comment", false, "仅粉丝可评论")

	d.AddCommand(create)
	return d
}
