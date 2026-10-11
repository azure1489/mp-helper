// Package analytics 把微信新版统计接口的原始返回整理成便于分析的扁平数据：
// 单篇文章表现（含推算送达人数、7 天数据、推送时刻）与每日粉丝增减（含发文日类型）。
package analytics

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/azure1489/mp-helper/internal/store"
	"github.com/azure1489/mp-helper/internal/types"
)

const (
	dateLayout   = "2006-01-02"
	minuteLayout = "2006-01-02 15:04"

	// MaxDays 是单次分析的最长日期跨度（含首尾）。
	MaxDays = 31

	apiTotalDetail = "getarticletotaldetail"
	apiBizSummary  = "getbizsummary"
	apiUserSummary = "getusersummary"
	apiUserCumul   = "getusercumulate"

	scenePush = "公众号消息"
	sceneAll  = "全部"

	// freepublish 最多翻 10 页（200 条），避免一次分析耗掉过多调用次数。
	publishedPageSize = 20
	publishedMaxPages = 10

	// 未定稿的单日统计缓存有效期；定稿后永久缓存。
	cacheTTL = time.Hour
)

// CST 为北京时间；统计日期与推送时刻都按它解释（固定时区，不依赖服务器 tzdata）。
var CST = time.FixedZone("CST", 8*3600)

// MinDate 是新版统计接口的数据起始日。
var MinDate = time.Date(2025, 11, 1, 0, 0, 0, 0, CST)

// Cache 缓存统计接口的单日响应。
type Cache interface {
	Get(api, date string) ([]byte, time.Time, bool)
	Put(api, date string, body []byte, at time.Time)
}

// Builder 汇总一个账号的数据。DataCube / ListPublished 调用微信（返回原始响应）；Cache 可为 nil。
type Builder struct {
	DataCube      func(api, begin, end string) (json.RawMessage, error)
	ListPublished func(offset, count int) (json.RawMessage, error)
	Meta          []store.ArticleMeta
	Cache         Cache
	Now           time.Time
}

// ValidateRange 解析并校验日期区间：不早于 MinDate、不晚于昨天（北京时间）、跨度不超过 MaxDays。
func ValidateRange(beginS, endS string, now time.Time) (time.Time, time.Time, error) {
	b, err1 := time.ParseInLocation(dateLayout, beginS, CST)
	e, err2 := time.ParseInLocation(dateLayout, endS, CST)
	if err1 != nil || err2 != nil {
		return b, e, errors.New("begin and end must be YYYY-MM-DD")
	}
	if e.Before(b) {
		return b, e, errors.New("end must not be before begin")
	}
	if b.Before(MinDate) {
		return b, e, fmt.Errorf("begin must be on or after %s (statistics start date)", MinDate.Format(dateLayout))
	}
	if yesterday := today(now).AddDate(0, 0, -1); e.After(yesterday) {
		return b, e, fmt.Errorf("end must be on or before yesterday (%s, Beijing time)", yesterday.Format(dateLayout))
	}
	if days(b, e) > MaxDays {
		return b, e, fmt.Errorf("date range must not exceed %d days", MaxDays)
	}
	return b, e, nil
}

func today(now time.Time) time.Time {
	y, m, d := now.In(CST).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, CST)
}

func days(b, e time.Time) int { return int(e.Sub(b).Hours()/24) + 1 }

func fmtDate(t time.Time) string { return t.Format(dateLayout) }

// ---- 微信原始返回 ----

type rawSource struct {
	UserCount int64  `json:"user_count"`
	SceneDesc string `json:"scene_desc"`
}

type rawDetail struct {
	StatDate          string      `json:"stat_date"`
	ReadUser          int64       `json:"read_user"`
	ReadUserSource    []rawSource `json:"read_user_source"`
	ShareUser         int64       `json:"share_user"`
	ZaikanUser        int64       `json:"zaikan_user"`
	LikeUser          int64       `json:"like_user"`
	CommentCount      int64       `json:"comment_count"`
	CollectionUser    int64       `json:"collection_user"`
	PraiseMoney       int64       `json:"praise_money"`
	ReadSubscribeUser int64       `json:"read_subscribe_user"`
	ReadDeliveryRate  *float64    `json:"read_delivery_rate"`
	ReadFinishRate    *float64    `json:"read_finish_rate"`
	ReadAvgActivetime *float64    `json:"read_avg_activetime"`
}

type rawArticle struct {
	RefDate     string      `json:"ref_date"`
	MsgID       string      `json:"msgid"`
	PublishType int         `json:"publish_type"`
	Title       string      `json:"title"`
	ContentURL  string      `json:"content_url"`
	DetailList  []rawDetail `json:"detail_list"`
}

type rawTotalDetail struct {
	List    []rawArticle `json:"list"`
	IsDelay bool         `json:"is_delay"`
}

type rawBizSummary struct {
	List []struct {
		RefDate string `json:"ref_date"`
		Detail  struct {
			ReadUser       int64       `json:"read_user"`
			ReadUserSource []rawSource `json:"read_user_source"`
			ShareUser      int64       `json:"share_user"`
			CollectionUser int64       `json:"collection_user"`
			SendPageCount  int64       `json:"send_page_count"`
		} `json:"detail"`
	} `json:"list"`
	IsDelay bool `json:"is_delay"`
}

type rawUserSummary struct {
	List []struct {
		RefDate    string `json:"ref_date"`
		UserSource int    `json:"user_source"`
		NewUser    int64  `json:"new_user"`
		CancelUser int64  `json:"cancel_user"`
	} `json:"list"`
}

type rawUserCumulate struct {
	List []struct {
		RefDate      string `json:"ref_date"`
		CumulateUser int64  `json:"cumulate_user"`
	} `json:"list"`
}

type rawPublished struct {
	TotalCount int `json:"total_count"`
	Item       []struct {
		UpdateTime int64 `json:"update_time"`
		Content    struct {
			NewsItem []struct {
				URL string `json:"url"`
			} `json:"news_item"`
		} `json:"content"`
	} `json:"item"`
}

var errcodeRe = regexp.MustCompile(`errcode=(\d+)`)

func warn(date, api string, err error) types.AnalyticsWarning {
	w := types.AnalyticsWarning{Date: date, API: api, Message: err.Error()}
	if m := errcodeRe.FindStringSubmatch(w.Message); len(m) == 2 {
		w.WechatErrcode, _ = strconv.ParseInt(m[1], 10, 64)
	}
	return w
}

// totalDetail 取某个发表日的 getarticletotaldetail（经缓存）。
// 定稿条件：该日发表的文章都已过 30 天统计期且统计日已早于前天，并且 is_delay 为 false。
func (b *Builder) totalDetail(d time.Time) (*rawTotalDetail, error) {
	ds := fmtDate(d)
	if b.Cache != nil {
		if body, at, ok := b.Cache.Get(apiTotalDetail, ds); ok {
			var r rawTotalDetail
			if json.Unmarshal(body, &r) == nil && (b.final(d, &r) || b.Now.Sub(at) < cacheTTL) {
				return &r, nil
			}
		}
	}
	raw, err := b.DataCube(apiTotalDetail, ds, ds)
	if err != nil {
		return nil, err
	}
	var r rawTotalDetail
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("%s %s: unexpected response: %w", apiTotalDetail, ds, err)
	}
	if b.Cache != nil {
		b.Cache.Put(apiTotalDetail, ds, raw, b.Now)
	}
	return &r, nil
}

func (b *Builder) final(d time.Time, r *rawTotalDetail) bool {
	return !r.IsDelay && !d.AddDate(0, 0, 29).After(today(b.Now).AddDate(0, 0, -2))
}

// ---- 单篇文章 ----

// Articles 汇总 [begin, end] 内发表的每篇文章。所有日期都取数失败时返回首个错误。
func (b *Builder) Articles(begin, end time.Time, series bool) (types.ArticlesReport, error) {
	rep := types.ArticlesReport{Begin: fmtDate(begin), End: fmtDate(end), Articles: []types.ArticleStats{}, Warnings: []types.AnalyticsWarning{}}
	var firstErr error
	failed := 0
	var raws []rawArticle
	for d := begin; !d.After(end); d = d.AddDate(0, 0, 1) {
		r, err := b.totalDetail(d)
		if err != nil {
			failed++
			if firstErr == nil {
				firstErr = err
			}
			rep.Warnings = append(rep.Warnings, warn(fmtDate(d), apiTotalDetail, err))
			continue
		}
		rep.IsDelay = rep.IsDelay || r.IsDelay
		raws = append(raws, r.List...)
	}
	if failed == days(begin, end) {
		return rep, firstErr
	}

	meta := make(map[string]store.ArticleMeta, len(b.Meta))
	for _, m := range b.Meta {
		meta[m.MsgID] = m
	}
	var published map[string]time.Time
	needPublished := false
	for _, a := range raws {
		if meta[a.MsgID].PublishedAt == "" {
			needPublished = true
			break
		}
	}
	if needPublished && b.ListPublished != nil {
		var w []types.AnalyticsWarning
		published, w = b.publishTimes()
		rep.Warnings = append(rep.Warnings, w...)
	}

	yesterday := today(b.Now).AddDate(0, 0, -1)
	for _, a := range raws {
		st := articleStats(a, yesterday, series)
		m := meta[a.MsgID]
		st.Category, st.Note = m.Category, m.Note
		if m.PublishedAt != "" {
			st.PublishTime, st.PublishTimeSource = strPtr(m.PublishedAt), strPtr("manual")
		} else if t, ok := published[a.MsgID]; ok && t.Format(dateLayout) == a.RefDate {
			// update_time 是最后编辑时间：只有落在发表当天才当作发布时刻。
			st.PublishTime, st.PublishTimeSource = strPtr(t.Format(minuteLayout)), strPtr("freepublish")
		}
		rep.Articles = append(rep.Articles, st)
	}
	sort.SliceStable(rep.Articles, func(i, j int) bool {
		x, y := rep.Articles[i], rep.Articles[j]
		if x.PublishDate != y.PublishDate {
			return x.PublishDate < y.PublishDate
		}
		return x.MsgID < y.MsgID
	})
	rep.ByCategory, rep.ByPublishHour, rep.MissingPublishTime = groupArticles(rep.Articles)
	return rep, nil
}

const uncategorized = "未分类"

// groupArticles 按分类与推送小时分组汇总。推送小时只取已通知且有推送时刻的文章。
func groupArticles(arts []types.ArticleStats) (byCat, byHour []types.GroupSummary, missing int) {
	cats := map[string][]types.ArticleStats{}
	hours := map[string][]types.ArticleStats{}
	for _, a := range arts {
		k := a.Category
		if k == "" {
			k = uncategorized
		}
		cats[k] = append(cats[k], a)
		if !a.Notified {
			continue
		}
		if a.PublishTime == nil || len(*a.PublishTime) < len(minuteLayout) {
			missing++
			continue
		}
		h := (*a.PublishTime)[11:13]
		hours[h] = append(hours[h], a)
	}
	byCat = groupSummaries(cats, func(a, b string) bool {
		if (a == uncategorized) != (b == uncategorized) {
			return b == uncategorized
		}
		return a < b
	})
	byHour = groupSummaries(hours, func(a, b string) bool { return a < b })
	return byCat, byHour, missing
}

func groupSummaries(groups map[string][]types.ArticleStats, less func(a, b string) bool) []types.GroupSummary {
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return less(keys[i], keys[j]) })
	out := make([]types.GroupSummary, 0, len(keys))
	for _, k := range keys {
		g := types.GroupSummary{Key: k, Articles: len(groups[k])}
		var read, push, share, coll, rate []float64
		for _, a := range groups[k] {
			if a.Day7 != nil {
				read = append(read, float64(a.Day7.ReadUser))
				push = append(push, float64(a.Day7.PushReadUser))
				share = append(share, float64(a.Day7.ShareUser))
				coll = append(coll, float64(a.Day7.CollectionUser))
			}
			if a.Notified {
				g.NotifiedArticles++
			}
			if a.PushOpenRate != nil {
				rate = append(rate, *a.PushOpenRate)
			}
		}
		g.Day7Articles = len(read)
		g.AvgDay7Read, g.MedianDay7Read = mean(read), median(read)
		g.AvgDay7PushRead, g.AvgDay7Share, g.AvgDay7Collection = mean(push), mean(share), mean(coll)
		g.AvgPushOpenRate, g.MedianPushOpenRate = mean(rate), median(rate)
		out = append(out, g)
	}
	return out
}

func mean(xs []float64) *float64 {
	if len(xs) == 0 {
		return nil
	}
	var sum float64
	for _, x := range xs {
		sum += x
	}
	return f64(sum / float64(len(xs)))
}

func median(xs []float64) *float64 {
	if len(xs) == 0 {
		return nil
	}
	ys := append([]float64(nil), xs...)
	sort.Float64s(ys)
	n := len(ys)
	if n%2 == 1 {
		return f64(ys[n/2])
	}
	return f64((ys[n/2-1] + ys[n/2]) / 2)
}

func articleStats(a rawArticle, yesterday time.Time, series bool) types.ArticleStats {
	st := types.ArticleStats{
		MsgID: a.MsgID, Title: a.Title, URL: a.ContentURL, PublishDate: a.RefDate,
		PublishType: a.PublishType, Notified: a.PublishType == 0,
	}
	snaps := make([]types.ArticleSnapshot, 0, len(a.DetailList))
	for _, d := range a.DetailList {
		snaps = append(snaps, snapshot(d))
	}
	sort.Slice(snaps, func(i, j int) bool { return snaps[i].StatDate < snaps[j].StatDate })
	if len(snaps) == 0 {
		return st
	}
	if snaps[0].StatDate == a.RefDate {
		st.Day0 = &snaps[0]
	}
	if pub, err := time.ParseInLocation(dateLayout, a.RefDate, CST); err == nil {
		// 累计值：取发表日 + 6 天当天或之前最近的一条；该日尚未出数（文章未满 7 天）则为 null。
		d7 := pub.AddDate(0, 0, 6)
		if !d7.After(yesterday) {
			for i := len(snaps) - 1; i >= 0; i-- {
				if snaps[i].StatDate <= fmtDate(d7) {
					st.Day7 = &snaps[i]
					break
				}
			}
		}
	}
	latest := snaps[len(snaps)-1]
	st.Latest = &latest
	if st.Notified && latest.ReadDeliveryRate != nil && *latest.ReadDeliveryRate > 0 {
		rate := *latest.ReadDeliveryRate
		st.PushOpenRate = &rate
		est := int64(math.Round(float64(latest.PushReadUser) / rate))
		st.DeliveredEst = &est
	}
	if series {
		st.Series = snaps
	}
	return st
}

func snapshot(d rawDetail) types.ArticleSnapshot {
	s := types.ArticleSnapshot{
		StatDate: d.StatDate, ReadUser: d.ReadUser, ReadBySource: sources(d.ReadUserSource),
		ShareUser: d.ShareUser, CollectionUser: d.CollectionUser, ZaikanUser: d.ZaikanUser,
		LikeUser: d.LikeUser, CommentCount: d.CommentCount, PraiseMoney: d.PraiseMoney,
		ReadSubscribeUser: d.ReadSubscribeUser, ReadDeliveryRate: d.ReadDeliveryRate,
		ReadFinishRate: d.ReadFinishRate, ReadAvgActivetime: d.ReadAvgActivetime,
	}
	s.PushReadUser = s.ReadBySource[scenePush]
	return s
}

func sources(src []rawSource) map[string]int64 {
	m := make(map[string]int64, len(src))
	for _, s := range src {
		if s.SceneDesc != sceneAll {
			m[s.SceneDesc] += s.UserCount
		}
	}
	return m
}

// publishTimes 翻 freepublish 列表，得到 msgid（mid_idx）→ update_time。
func (b *Builder) publishTimes() (map[string]time.Time, []types.AnalyticsWarning) {
	out := map[string]time.Time{}
	for page := 0; page < publishedMaxPages; page++ {
		raw, err := b.ListPublished(page*publishedPageSize, publishedPageSize)
		if err != nil {
			return out, []types.AnalyticsWarning{warn("", "freepublish/batchget", err)}
		}
		var r rawPublished
		if err := json.Unmarshal(raw, &r); err != nil {
			return out, []types.AnalyticsWarning{warn("", "freepublish/batchget", err)}
		}
		for _, it := range r.Item {
			for _, n := range it.Content.NewsItem {
				if id := msgIDFromURL(n.URL); id != "" {
					out[id] = time.Unix(it.UpdateTime, 0).In(CST)
				}
			}
		}
		if (page+1)*publishedPageSize >= r.TotalCount || len(r.Item) == 0 {
			return out, nil
		}
	}
	return out, []types.AnalyticsWarning{{API: "freepublish/batchget",
		Message: fmt.Sprintf("only the latest %d published articles were scanned for publish times", publishedPageSize*publishedMaxPages)}}
}

// msgIDFromURL 从文章链接的 mid、idx 参数得到与统计接口一致的 msgid（"<mid>_<idx>"）。
func msgIDFromURL(s string) string {
	u, err := url.Parse(strings.ReplaceAll(s, "&amp;", "&"))
	if err != nil {
		return ""
	}
	q := u.Query()
	mid, idx := q.Get("mid"), q.Get("idx")
	if mid == "" || idx == "" {
		return ""
	}
	return mid + "_" + idx
}

func strPtr(s string) *string { return &s }

// ---- 每日 ----

// rangeRaw 是按日期区间返回的统计接口（getusersummary / getusercumulate / getbizsummary）的通用外形。
type rangeRaw struct {
	List    []json.RawMessage `json:"list"`
	IsDelay bool              `json:"is_delay"`
}

// rangeFinal：统计日早于前天的数据视为定稿（次日 8 点后才出数，留一天余量）。
func (b *Builder) rangeFinal(d time.Time) bool {
	return !d.After(today(b.Now).AddDate(0, 0, -2))
}

// cachedRange 取 [begin, end] 每天的响应（只含该日的 list 项）：定稿或未过期的日期读缓存，
// 其余按连续区间、每段最多 chunk 天调用微信，再按 ref_date 拆开逐日写回缓存。返回调用次数。
func (b *Builder) cachedRange(api string, begin, end time.Time, chunk int, fail func(label, api string, err error)) (map[string][]byte, int) {
	out := map[string][]byte{}
	var missing []time.Time
	for d := begin; !d.After(end); d = d.AddDate(0, 0, 1) {
		ds := fmtDate(d)
		if b.Cache != nil {
			if body, at, ok := b.Cache.Get(api, ds); ok {
				var r rangeRaw
				if json.Unmarshal(body, &r) == nil && ((!r.IsDelay && b.rangeFinal(d)) || b.Now.Sub(at) < cacheTTL) {
					out[ds] = body
					continue
				}
			}
		}
		missing = append(missing, d)
	}
	calls := 0
	for _, c := range runs(missing, chunk) {
		calls++
		raw, err := b.DataCube(api, fmtDate(c[0]), fmtDate(c[1]))
		var r rangeRaw
		if err == nil {
			err = json.Unmarshal(raw, &r)
		}
		if err != nil {
			fail(fmtDate(c[0])+"~"+fmtDate(c[1]), api, err)
			continue
		}
		per := map[string][]json.RawMessage{}
		for _, it := range r.List {
			var k struct {
				RefDate string `json:"ref_date"`
			}
			if json.Unmarshal(it, &k) == nil {
				per[k.RefDate] = append(per[k.RefDate], it)
			}
		}
		for d := c[0]; !d.After(c[1]); d = d.AddDate(0, 0, 1) {
			ds := fmtDate(d)
			items := per[ds]
			if items == nil {
				items = []json.RawMessage{}
			}
			body, _ := json.Marshal(rangeRaw{List: items, IsDelay: r.IsDelay})
			out[ds] = body
			if b.Cache != nil {
				b.Cache.Put(api, ds, body, b.Now)
			}
		}
	}
	return out, calls
}

// runs 把升序日期切成连续区间，每段最多 size 天。
func runs(ds []time.Time, size int) [][2]time.Time {
	var out [][2]time.Time
	for i := 0; i < len(ds); {
		j := i
		for j+1 < len(ds) && j+1-i < size && ds[j+1].Equal(ds[j].AddDate(0, 0, 1)) {
			j++
		}
		out = append(out, [2]time.Time{ds[i], ds[j]})
		i = j + 1
	}
	return out
}

// cadenceOf 由近 7 天推送天数判定发文节奏。
func cadenceOf(n int) string {
	switch {
	case n >= 6:
		return "daily"
	case n >= 3:
		return "every_other_day"
	case n >= 1:
		return "sparse"
	}
	return "none"
}

// Daily 汇总 [begin, end] 每天的粉丝增减、阅读汇总与发文情况。所有接口都失败时返回首个错误。
func (b *Builder) Daily(begin, end time.Time) (types.DailyReport, error) {
	rep := types.DailyReport{Begin: fmtDate(begin), End: fmtDate(end), Days: []types.DailyStats{}, Summary: []types.DayTypeSummary{},
		ByCadence: []types.CadenceSummary{}, CadencePeriods: []types.CadencePeriod{}, Warnings: []types.AnalyticsWarning{}}
	clamp := func(d time.Time) time.Time {
		if d.Before(MinDate) {
			return MinDate
		}
		return d
	}
	// 多取前一天算推送次日与取关率（÷ 前一天累计关注）；推送情况再往前多取 6 天算近 7 天节奏。
	from := clamp(begin.AddDate(0, 0, -1))
	pushFrom := clamp(begin.AddDate(0, 0, -6))
	var firstErr error
	calls, failed := 0, 0
	fail := func(label, api string, err error) {
		failed++
		if firstErr == nil {
			firstErr = err
		}
		rep.Warnings = append(rep.Warnings, warn(label, api, err))
	}

	pushed := map[string]int{}
	published := map[string]int{}
	msgids := map[string][]string{}
	known := map[string]bool{} // 推送情况已知的日期
	for d := pushFrom; !d.After(end); d = d.AddDate(0, 0, 1) {
		calls++
		r, err := b.totalDetail(d)
		if err != nil {
			fail(fmtDate(d), apiTotalDetail, err)
			continue
		}
		rep.IsDelay = rep.IsDelay || r.IsDelay
		ds := fmtDate(d)
		known[ds] = true
		for _, a := range r.List {
			published[ds]++
			if a.PublishType == 0 {
				pushed[ds]++
			}
			msgids[ds] = append(msgids[ds], a.MsgID)
		}
	}

	userBodies, n := b.cachedRange(apiUserSummary, begin, end, 7, fail)
	calls += n
	cumulBodies, n := b.cachedRange(apiUserCumul, from, end, 7, fail)
	calls += n
	bizBodies, n := b.cachedRange(apiBizSummary, begin, end, 30, fail)
	calls += n
	if failed > 0 && failed == calls {
		return rep, firstErr
	}

	newU, cancelU := map[string]int64{}, map[string]int64{}
	newBySrc := map[string]map[string]int64{}
	hasSummary := map[string]bool{}
	for _, body := range userBodies {
		var r rawUserSummary
		if json.Unmarshal(body, &r) != nil {
			continue
		}
		for _, x := range r.List {
			hasSummary[x.RefDate] = true
			newU[x.RefDate] += x.NewUser
			cancelU[x.RefDate] += x.CancelUser
			if x.NewUser > 0 {
				if newBySrc[x.RefDate] == nil {
					newBySrc[x.RefDate] = map[string]int64{}
				}
				newBySrc[x.RefDate][strconv.Itoa(x.UserSource)] += x.NewUser
			}
		}
	}

	cumul := map[string]int64{}
	for _, body := range cumulBodies {
		var r rawUserCumulate
		if json.Unmarshal(body, &r) != nil {
			continue
		}
		for _, x := range r.List {
			cumul[x.RefDate] += x.CumulateUser
		}
	}

	type biz struct {
		read, share, collection, send int64
		src                           map[string]int64
	}
	bizByDate := map[string]biz{}
	for _, body := range bizBodies {
		var r rawBizSummary
		if json.Unmarshal(body, &r) != nil {
			continue
		}
		rep.IsDelay = rep.IsDelay || r.IsDelay
		for _, x := range r.List {
			bizByDate[x.RefDate] = biz{x.Detail.ReadUser, x.Detail.ShareUser, x.Detail.CollectionUser, x.Detail.SendPageCount, sources(x.Detail.ReadUserSource)}
		}
	}

	for d := begin; !d.After(end); d = d.AddDate(0, 0, 1) {
		ds, prev := fmtDate(d), fmtDate(d.AddDate(0, 0, -1))
		day := types.DailyStats{Date: ds, PushedCount: pushed[ds], PublishedCount: published[ds], MsgIDs: msgids[ds], NewBySource: newBySrc[ds]}
		if day.MsgIDs == nil {
			day.MsgIDs = []string{}
		}
		switch {
		case pushed[ds] > 0:
			day.DayType = "pushed"
		case pushed[prev] > 0:
			day.DayType = "day_after_push"
		default:
			day.DayType = "quiet"
		}
		if w := d.AddDate(0, 0, -6); !w.Before(MinDate) {
			n, complete := 0, true
			for x := w; !x.After(d); x = x.AddDate(0, 0, 1) {
				complete = complete && known[fmtDate(x)]
				if pushed[fmtDate(x)] > 0 {
					n++
				}
			}
			if complete {
				day.PushDays7d = &n
				day.Cadence = cadenceOf(n)
			}
		}
		c, hasCumul := cumul[ds]
		if hasCumul {
			day.CumulateUser = i64(c)
		}
		// getusersummary 只返回有增减的来源：当天有累计数据却无增减记录，视为 0。
		if hasSummary[ds] || hasCumul {
			day.NewUser, day.CancelUser = i64(newU[ds]), i64(cancelU[ds])
			day.NetUser = i64(newU[ds] - cancelU[ds])
			if p, ok := cumul[prev]; ok && p > 0 {
				r := float64(cancelU[ds]) / float64(p)
				day.CancelRate = &r
			}
		} else {
			rep.Warnings = append(rep.Warnings, types.AnalyticsWarning{Date: ds, API: apiUserSummary, Message: "no user data for this date yet"})
		}
		if x, ok := bizByDate[ds]; ok {
			day.ReadUser, day.ShareUser, day.CollectionUser, day.SendPageCount = i64(x.read), i64(x.share), i64(x.collection), i64(x.send)
			day.ReadBySource = x.src
		}
		rep.Days = append(rep.Days, day)
	}

	for _, dt := range []string{"pushed", "day_after_push", "quiet"} {
		a := average(rep.Days, func(d types.DailyStats) bool { return d.DayType == dt })
		rep.Summary = append(rep.Summary, types.DayTypeSummary{DayType: dt, Days: a.days, AvgNewUser: a.new, AvgCancelUser: a.cancel, AvgCancelRate: a.rate})
	}
	for _, cd := range []string{"daily", "every_other_day", "sparse", "none"} {
		a := average(rep.Days, func(d types.DailyStats) bool { return d.Cadence == cd })
		rep.ByCadence = append(rep.ByCadence, types.CadenceSummary{Cadence: cd, Days: a.days, AvgNewUser: a.new, AvgCancelUser: a.cancel, AvgCancelRate: a.rate})
	}
	for i := 0; i < len(rep.Days); {
		j := i
		for j+1 < len(rep.Days) && rep.Days[j+1].Cadence == rep.Days[i].Cadence {
			j++
		}
		if cd := rep.Days[i].Cadence; cd != "" {
			run := rep.Days[i : j+1]
			a := average(run, func(types.DailyStats) bool { return true })
			rep.CadencePeriods = append(rep.CadencePeriods, types.CadencePeriod{Cadence: cd, Begin: run[0].Date, End: run[len(run)-1].Date,
				Days: a.days, AvgNewUser: a.new, AvgCancelUser: a.cancel, AvgCancelRate: a.rate})
		}
		i = j + 1
	}
	return rep, nil
}

type avgStats struct {
	days              int
	new, cancel, rate *float64
}

// average 对满足条件的日期求平均新增、取关（有用户数据的日子）与取关率（有取关率的日子）。
func average(ds []types.DailyStats, keep func(types.DailyStats) bool) avgStats {
	var a avgStats
	var news, cancels, rates []float64
	for _, d := range ds {
		if !keep(d) {
			continue
		}
		a.days++
		if d.NewUser != nil {
			news = append(news, float64(*d.NewUser))
			cancels = append(cancels, float64(*d.CancelUser))
		}
		if d.CancelRate != nil {
			rates = append(rates, *d.CancelRate)
		}
	}
	a.new, a.cancel, a.rate = mean(news), mean(cancels), mean(rates)
	return a
}

func i64(v int64) *int64     { return &v }
func f64(v float64) *float64 { return &v }
