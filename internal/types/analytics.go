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

// GroupSummary 是一组文章（同一分类或同一推送小时）的平均表现。
// 7 天指标只统计已满 7 天的文章；打开率只统计已通知且有送达率的文章。
type GroupSummary struct {
	Key                string   `json:"key"`
	Articles           int      `json:"articles"`
	Day7Articles       int      `json:"day7_articles"`
	AvgDay7Read        *float64 `json:"avg_day7_read"`
	MedianDay7Read     *float64 `json:"median_day7_read"`
	AvgDay7PushRead    *float64 `json:"avg_day7_push_read"`
	AvgDay7Share       *float64 `json:"avg_day7_share"`
	AvgDay7Collection  *float64 `json:"avg_day7_collection"`
	NotifiedArticles   int      `json:"notified_articles"`
	AvgPushOpenRate    *float64 `json:"avg_push_open_rate"`
	MedianPushOpenRate *float64 `json:"median_push_open_rate"`
}

// ArticlesReport 是 GET /api/v1/analytics/articles 的返回。
type ArticlesReport struct {
	Begin    string         `json:"begin"`
	End      string         `json:"end"`
	IsDelay  bool           `json:"is_delay"`
	Articles []ArticleStats `json:"articles"`
	// ByCategory 按补录的分类分组（未补录的归入「未分类」）。
	ByCategory []GroupSummary `json:"by_category"`
	// ByPublishHour 按推送时刻的小时（"00"–"23"）分组，只含已通知且有推送时刻的文章。
	ByPublishHour []GroupSummary `json:"by_publish_hour"`
	// MissingPublishTime 是已通知但没有推送时刻、未计入 ByPublishHour 的篇数。
	MissingPublishTime int                `json:"missing_publish_time"`
	Warnings           []AnalyticsWarning `json:"warnings"`
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
	// PushDays7d 是截至当天的近 7 天（含当天）有推送的天数；窗口有缺数时为 null。
	PushDays7d *int `json:"push_days_7d"`
	// Cadence 由 PushDays7d 判定：daily（≥6）、every_other_day（3–5）、sparse（1–2）、none（0）；未知为空。
	Cadence string `json:"cadence,omitempty"`
}

// DayTypeSummary 按 day_type 汇总的平均粉丝增减。
type DayTypeSummary struct {
	DayType       string   `json:"day_type"`
	Days          int      `json:"days"`
	AvgNewUser    *float64 `json:"avg_new_user"`
	AvgCancelUser *float64 `json:"avg_cancel_user"`
	AvgCancelRate *float64 `json:"avg_cancel_rate"`
}

// CadenceSummary 按发文节奏汇总的平均粉丝增减。
type CadenceSummary struct {
	Cadence       string   `json:"cadence"`
	Days          int      `json:"days"`
	AvgNewUser    *float64 `json:"avg_new_user"`
	AvgCancelUser *float64 `json:"avg_cancel_user"`
	AvgCancelRate *float64 `json:"avg_cancel_rate"`
}

// CadencePeriod 是一段节奏相同的连续日期。
type CadencePeriod struct {
	Cadence       string   `json:"cadence"`
	Begin         string   `json:"begin"`
	End           string   `json:"end"`
	Days          int      `json:"days"`
	AvgNewUser    *float64 `json:"avg_new_user"`
	AvgCancelUser *float64 `json:"avg_cancel_user"`
	AvgCancelRate *float64 `json:"avg_cancel_rate"`
}

// DailyReport 是 GET /api/v1/analytics/daily 的返回。
type DailyReport struct {
	Begin          string             `json:"begin"`
	End            string             `json:"end"`
	IsDelay        bool               `json:"is_delay"`
	Days           []DailyStats       `json:"days"`
	Summary        []DayTypeSummary   `json:"summary"`
	ByCadence      []CadenceSummary   `json:"by_cadence"`
	CadencePeriods []CadencePeriod    `json:"cadence_periods"`
	Warnings       []AnalyticsWarning `json:"warnings"`
}

// ArchiveAccountResult 是一次存档中单个账号的结果。
type ArchiveAccountResult struct {
	AccountID int64  `json:"account_id"`
	Name      string `json:"name"`
	Calls     int    `json:"calls"`             // 实际调用微信的次数
	Failed    int    `json:"failed"`            // 其中失败的次数
	Stopped   string `json:"stopped,omitempty"` // 提前停止的原因（次数用完、无权限、配额耗尽）
}

// ArchiveStatus 是 GET /admin/archive 的返回。
type ArchiveStatus struct {
	Running      bool                   `json:"running"`
	LastStarted  string                 `json:"last_started,omitempty"`
	LastFinished string                 `json:"last_finished,omitempty"`
	Results      []ArchiveAccountResult `json:"results"`
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
