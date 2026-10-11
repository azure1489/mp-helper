package types

// ---- 数据分析（/api/v1/analytics/*）与文章补录信息（/api/v1/articles/*）----

// ArticleSnapshot 是单篇文章截至 StatDate 的累计数据（发表日至统计日的新增总和）。
type ArticleSnapshot struct {
	StatDate          string           `json:"stat_date"`
	ReadUser          int64            `json:"read_user"`
	PushReadUser      int64            `json:"push_read_user"` // 来源为「公众号消息」的阅读人数
	ReadBySource      map[string]int64 `json:"read_by_source"` // 不含「全部」
	ShareUser         int64            `json:"share_user"`
	CollectionUser    int64            `json:"collection_user"`
	ZaikanUser        int64            `json:"zaikan_user"`
	LikeUser          int64            `json:"like_user"`
	CommentCount      int64            `json:"comment_count"`
	PraiseMoney       int64            `json:"praise_money"` // 分
	ReadSubscribeUser int64            `json:"read_subscribe_user"`
	ReadDeliveryRate  *float64         `json:"read_delivery_rate"`
	ReadFinishRate    *float64         `json:"read_finish_rate"`
	ReadAvgActivetime *float64         `json:"read_avg_activetime"` // 分钟
}

// ArticleStats 是一篇文章的表现汇总。
type ArticleStats struct {
	MsgID       string `json:"msgid"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	PublishDate string `json:"publish_date"`
	PublishType int    `json:"publish_type"` // 0 已通知（推送），1 未开启通知
	Notified    bool   `json:"notified"`
	// PublishTime 为北京时间 "YYYY-MM-DD HH:MM"；来源 manual（人工补录）或 freepublish（发布记录），未知时为 null。
	PublishTime       *string `json:"publish_time"`
	PublishTimeSource *string `json:"publish_time_source"`
	Category          string  `json:"category,omitempty"`
	Note              string  `json:"note,omitempty"`
	// DeliveredEst = 公众号消息阅读人数 ÷ 阅读送达率（推算值）；未通知或送达率为 0 时为 null。
	DeliveredEst *int64 `json:"delivered_est"`
	// PushOpenRate 即最新一天的 read_delivery_rate（公众号消息阅读人数 ÷ 送达人数）。
	PushOpenRate *float64          `json:"push_open_rate"`
	Day0         *ArticleSnapshot  `json:"day0"`
	Day7         *ArticleSnapshot  `json:"day7"` // 截至发表日 + 6 天；未满 7 天为 null
	Latest       *ArticleSnapshot  `json:"latest"`
	Series       []ArticleSnapshot `json:"series,omitempty"`
}

// AnalyticsWarning 记录某一天（或某段日期）取数失败或缺数，不影响其余结果。
type AnalyticsWarning struct {
	Date          string `json:"date"`
	API           string `json:"api"`
	Message       string `json:"message"`
	WechatErrcode int64  `json:"wechat_errcode,omitempty"`
}

// ArticlesReport 是 GET /api/v1/analytics/articles 的返回。
type ArticlesReport struct {
	Begin    string             `json:"begin"`
	End      string             `json:"end"`
	IsDelay  bool               `json:"is_delay"`
	Articles []ArticleStats     `json:"articles"`
	Warnings []AnalyticsWarning `json:"warnings"`
}

// DailyStats 是账号某一天的粉丝增减、阅读汇总与发文情况。
type DailyStats struct {
	Date         string           `json:"date"`
	DayType      string           `json:"day_type"` // pushed | day_after_push | quiet
	NewUser      *int64           `json:"new_user"`
	CancelUser   *int64           `json:"cancel_user"`
	NetUser      *int64           `json:"net_user"`
	NewBySource  map[string]int64 `json:"new_by_source,omitempty"` // 键为微信 user_source 编码
	CumulateUser *int64           `json:"cumulate_user"`
	// CancelRate = 当天取关 ÷ 前一天累计关注。
	CancelRate     *float64         `json:"cancel_rate"`
	ReadUser       *int64           `json:"read_user"`
	ReadBySource   map[string]int64 `json:"read_by_source,omitempty"`
	ShareUser      *int64           `json:"share_user"`
	CollectionUser *int64           `json:"collection_user"`
	SendPageCount  *int64           `json:"send_page_count"`
	PushedCount    int              `json:"pushed_count"`    // 当天发表且已通知的篇数
	PublishedCount int              `json:"published_count"` // 当天发表的篇数（含未通知）
	MsgIDs         []string         `json:"msgids"`
}

// DayTypeSummary 按 day_type 汇总的平均粉丝增减。
type DayTypeSummary struct {
	DayType       string   `json:"day_type"`
	Days          int      `json:"days"`
	AvgNewUser    *float64 `json:"avg_new_user"`
	AvgCancelUser *float64 `json:"avg_cancel_user"`
	AvgCancelRate *float64 `json:"avg_cancel_rate"`
}

// DailyReport 是 GET /api/v1/analytics/daily 的返回。
type DailyReport struct {
	Begin    string             `json:"begin"`
	End      string             `json:"end"`
	IsDelay  bool               `json:"is_delay"`
	Days     []DailyStats       `json:"days"`
	Summary  []DayTypeSummary   `json:"summary"`
	Warnings []AnalyticsWarning `json:"warnings"`
}

// ArticleMetaRequest 是 PUT /api/v1/articles/:msgid/meta 的请求体；省略的字段保留原值。
type ArticleMetaRequest struct {
	PublishedAt *string `json:"published_at,omitempty"` // "YYYY-MM-DD HH:MM"（北京时间）
	Category    *string `json:"category,omitempty"`
	Note        *string `json:"note,omitempty"`
}

// ArticleMeta 是一条文章补录信息。
type ArticleMeta struct {
	MsgID       string `json:"msgid"`
	PublishedAt string `json:"published_at"`
	Category    string `json:"category"`
	Note        string `json:"note"`
	UpdatedAt   string `json:"updated_at"`
}
