package cli

import "github.com/spf13/cobra"

// 只读查询：已发布图文、草稿读取、数据统计。输出为微信原始返回。

type pageFlags struct {
	offset, count int
	withContent   bool
}

func (p *pageFlags) bind(cmd *cobra.Command) {
	f := cmd.Flags()
	f.IntVar(&p.offset, "offset", 0, "起始位置")
	f.IntVar(&p.count, "count", 20, "条数（1–20）")
	f.BoolVar(&p.withContent, "with-content", false, "同时返回正文（缺省只返回标题、链接等字段）")
}

func newPublishedCmd() *cobra.Command {
	p := &cobra.Command{Use: "published", Short: "已发布图文（只读）"}

	var pf pageFlags
	list := &cobra.Command{
		Use:   "list",
		Short: "分页列出已发布图文（「群发」的图文不在其中，可用 stats getarticletotaldetail 按日期查到）",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			res, err := c.ListPublished(pf.offset, pf.count, pf.withContent)
			if err != nil {
				return err
			}
			return printJSON(cmd, res)
		},
	}
	pf.bind(list)

	get := &cobra.Command{
		Use:   "get <article_id>",
		Short: "读取一篇已发布图文（含正文）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			res, err := c.GetPublished(args[0])
			if err != nil {
				return err
			}
			return printJSON(cmd, res)
		},
	}

	p.AddCommand(list, get)
	return p
}

func newDraftListCmd() *cobra.Command {
	var pf pageFlags
	list := &cobra.Command{
		Use:   "list",
		Short: "分页列出草稿箱",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			res, err := c.ListDrafts(pf.offset, pf.count, pf.withContent)
			if err != nil {
				return err
			}
			return printJSON(cmd, res)
		},
	}
	pf.bind(list)
	return list
}

func newDraftGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <media_id>",
		Short: "读取一篇草稿（保存后回读核对用）",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			res, err := c.GetDraft(args[0])
			if err != nil {
				return err
			}
			return printJSON(cmd, res)
		},
	}
}

func newStatsCmd() *cobra.Command {
	var begin, end string
	st := &cobra.Command{
		Use:   "stats <api>",
		Short: "数据统计原样透传：getarticletotaldetail | getarticleread | getarticleshare | getbizsummary | getusersummary | getusercumulate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			res, err := c.DataCube(args[0], begin, end)
			if err != nil {
				return err
			}
			return printJSON(cmd, res)
		},
	}
	st.Flags().StringVar(&begin, "begin", "", "开始日期 YYYY-MM-DD（必填）")
	st.Flags().StringVar(&end, "end", "", "结束日期 YYYY-MM-DD（必填）")
	_ = st.MarkFlagRequired("begin")
	_ = st.MarkFlagRequired("end")
	return st
}
