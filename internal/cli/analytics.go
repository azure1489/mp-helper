package cli

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/azure1489/mp-helper/internal/types"
)

// 数据分析（单篇表现、每日粉丝增减）与文章补录信息（推送时刻、分类）。

// sceneCols 是 CSV 中固定展开的阅读来源列（不含「公众号消息」，它单列为 push_read_user）。
var sceneCols = []string{"聊天会话", "朋友圈", "公众号主页", "推荐", "搜一搜", "其他"}

type rangeFlags struct {
	begin, end, format string
}

func (r *rangeFlags) bind(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringVar(&r.begin, "begin", "", "开始日期 YYYY-MM-DD（必填，不早于 2025-11-01）")
	f.StringVar(&r.end, "end", "", "结束日期 YYYY-MM-DD（必填，最晚昨天，跨度 ≤ 31 天）")
	f.StringVar(&r.format, "format", "json", "输出格式：json | csv")
	_ = cmd.MarkFlagRequired("begin")
	_ = cmd.MarkFlagRequired("end")
}

func (r *rangeFlags) check() error {
	if r.format != "json" && r.format != "csv" {
		return fmt.Errorf("--format must be json or csv")
	}
	return nil
}

func newAnalyticsCmd() *cobra.Command {
	a := &cobra.Command{Use: "analytics", Short: "数据分析：单篇文章表现、每日粉丝增减与发文情况"}

	var af rangeFlags
	var series bool
	articles := &cobra.Command{
		Use:   "articles",
		Short: "按发表日期汇总每篇文章：当天 / 7 天 / 最新的阅读、来源、分享、收藏，推送打开率、推算送达人数、推送时刻",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := af.check(); err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			rep, err := c.AnalyticsArticles(af.begin, af.end, series)
			if err != nil {
				return err
			}
			if af.format == "csv" {
				printWarnings(cmd, rep.Warnings)
				return writeArticlesCSV(cmd.OutOrStdout(), rep.Articles)
			}
			return printJSON(cmd, rep)
		},
	}
	af.bind(articles)
	articles.Flags().BoolVar(&series, "series", false, "附每篇的逐日累计明细（仅 json）")

	var df rangeFlags
	daily := &cobra.Command{
		Use:   "daily",
		Short: "每天的新增 / 取关 / 累计关注、取关率、阅读汇总与来源、当天推送篇数，并按推送日 / 推送次日 / 不发日汇总",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := df.check(); err != nil {
				return err
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			rep, err := c.AnalyticsDaily(df.begin, df.end)
			if err != nil {
				return err
			}
			if df.format == "csv" {
				printWarnings(cmd, rep.Warnings)
				return writeDailyCSV(cmd.OutOrStdout(), rep.Days)
			}
			return printJSON(cmd, rep)
		},
	}
	df.bind(daily)

	a.AddCommand(articles, daily)
	return a
}

func newArticleCmd() *cobra.Command {
	a := &cobra.Command{Use: "article", Short: "文章补录信息（推送时刻、分类、备注），只存本服务，不改公众号"}
	meta := &cobra.Command{Use: "meta", Short: "补录或查看文章信息"}

	var publishedAt, category, note string
	set := &cobra.Command{
		Use:   "set <msgid>",
		Short: "补录一篇文章（msgid 取自 analytics articles，如 2247549594_1）；只更新传入的字段",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var req types.ArticleMetaRequest
			f := cmd.Flags()
			if f.Changed("published-at") {
				req.PublishedAt = &publishedAt
			}
			if f.Changed("category") {
				req.Category = &category
			}
			if f.Changed("note") {
				req.Note = &note
			}
			if req.PublishedAt == nil && req.Category == nil && req.Note == nil {
				return fmt.Errorf("set at least one of --published-at, --category, --note")
			}
			c, err := newClient()
			if err != nil {
				return err
			}
			res, err := c.SetArticleMeta(args[0], req)
			if err != nil {
				return err
			}
			return printJSON(cmd, res)
		},
	}
	set.Flags().StringVar(&publishedAt, "published-at", "", `推送时刻，北京时间 "YYYY-MM-DD HH:MM"`)
	set.Flags().StringVar(&category, "category", "", "分类（如 攻略 / 快讯 / 专题）")
	set.Flags().StringVar(&note, "note", "", "备注")

	list := &cobra.Command{
		Use:   "list",
		Short: "列出全部补录信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newClient()
			if err != nil {
				return err
			}
			res, err := c.ListArticleMeta()
			if err != nil {
				return err
			}
			return printJSON(cmd, res)
		},
	}

	meta.AddCommand(set, list)
	a.AddCommand(meta)
	return a
}

// printWarnings 在 CSV 模式下把取数告警写到 stderr，避免混进表格。
func printWarnings(cmd *cobra.Command, ws []types.AnalyticsWarning) {
	for _, w := range ws {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s %s: %s\n", w.Date, w.API, w.Message)
	}
}

func fmtInt(p *int64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatInt(*p, 10)
}

func fmtFloat(p *float64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatFloat(*p, 'f', -1, 64)
}

func fmtStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

var snapshotCols = []string{"stat_date", "read_user", "push_read_user", "share_user", "collection_user", "zaikan_user",
	"like_user", "comment_count", "read_subscribe_user", "read_delivery_rate", "read_finish_rate", "read_avg_activetime"}

func snapshotRow(s *types.ArticleSnapshot) []string {
	if s == nil {
		return make([]string, len(snapshotCols)+len(sceneCols))
	}
	n := func(v int64) string { return strconv.FormatInt(v, 10) }
	row := []string{s.StatDate, n(s.ReadUser), n(s.PushReadUser), n(s.ShareUser), n(s.CollectionUser), n(s.ZaikanUser),
		n(s.LikeUser), n(s.CommentCount), n(s.ReadSubscribeUser), fmtFloat(s.ReadDeliveryRate), fmtFloat(s.ReadFinishRate), fmtFloat(s.ReadAvgActivetime)}
	for _, sc := range sceneCols {
		row = append(row, n(s.ReadBySource[sc]))
	}
	return row
}

func writeArticlesCSV(w io.Writer, arts []types.ArticleStats) error {
	cw := csv.NewWriter(w)
	header := []string{"msgid", "title", "publish_date", "publish_type", "notified", "publish_time", "publish_time_source",
		"category", "note", "delivered_est", "push_open_rate"}
	for _, p := range []string{"day0", "day7", "latest"} {
		for _, c := range snapshotCols {
			header = append(header, p+"_"+c)
		}
		for _, sc := range sceneCols {
			header = append(header, p+"_src_"+sc)
		}
	}
	header = append(header, "url")
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, a := range arts {
		row := []string{a.MsgID, a.Title, a.PublishDate, strconv.Itoa(a.PublishType), strconv.FormatBool(a.Notified),
			fmtStr(a.PublishTime), fmtStr(a.PublishTimeSource), a.Category, a.Note, fmtInt(a.DeliveredEst), fmtFloat(a.PushOpenRate)}
		row = append(row, snapshotRow(a.Day0)...)
		row = append(row, snapshotRow(a.Day7)...)
		row = append(row, snapshotRow(a.Latest)...)
		row = append(row, a.URL)
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

func writeDailyCSV(w io.Writer, ds []types.DailyStats) error {
	cw := csv.NewWriter(w)
	header := []string{"date", "day_type", "pushed_count", "published_count", "new_user", "cancel_user", "net_user",
		"cumulate_user", "cancel_rate", "read_user", "push_read_user"}
	for _, sc := range sceneCols {
		header = append(header, "src_"+sc)
	}
	header = append(header, "share_user", "collection_user", "send_page_count", "new_by_source", "msgids")
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, d := range ds {
		src := func(k string) string {
			if d.ReadBySource == nil {
				return ""
			}
			return strconv.FormatInt(d.ReadBySource[k], 10)
		}
		row := []string{d.Date, d.DayType, strconv.Itoa(d.PushedCount), strconv.Itoa(d.PublishedCount), fmtInt(d.NewUser),
			fmtInt(d.CancelUser), fmtInt(d.NetUser), fmtInt(d.CumulateUser), fmtFloat(d.CancelRate), fmtInt(d.ReadUser), src("公众号消息")}
		for _, sc := range sceneCols {
			row = append(row, src(sc))
		}
		keys := make([]string, 0, len(d.NewBySource))
		for k := range d.NewBySource {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k+":"+strconv.FormatInt(d.NewBySource[k], 10))
		}
		row = append(row, fmtInt(d.ShareUser), fmtInt(d.CollectionUser), fmtInt(d.SendPageCount), strings.Join(parts, ";"), strings.Join(d.MsgIDs, ";"))
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
