package analytics

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/azure1489/mp-helper/internal/store"
	"github.com/azure1489/mp-helper/internal/types"
)

// now = 2026-10-12 10:00 北京时间，昨天为 2026-10-11。
var now = time.Date(2026, 10, 12, 10, 0, 0, 0, CST)

func day(s string) time.Time {
	t, _ := time.ParseInLocation(dateLayout, s, CST)
	return t
}

func detail(stat string, read, push int64, rate float64) string {
	return fmt.Sprintf(`{"stat_date":%q,"read_user":%d,"read_user_source":[{"user_count":%d,"scene_desc":"全部"},{"user_count":%d,"scene_desc":"公众号消息"},{"user_count":%d,"scene_desc":"推荐"}],"share_user":3,"collection_user":2,"read_delivery_rate":%v}`,
		stat, read, read, push, read-push, rate)
}

// fakeCube 按 api 与日期返回预置响应，记录调用。
type fakeCube struct {
	resp  map[string]string // key: api + " " + begin
	errs  map[string]error
	calls []string
}

func (f *fakeCube) DataCube(api, begin, end string) (json.RawMessage, error) {
	f.calls = append(f.calls, api+" "+begin+" "+end)
	k := api + " " + begin
	if err := f.errs[k]; err != nil {
		return nil, err
	}
	if r, ok := f.resp[k]; ok {
		return json.RawMessage(r), nil
	}
	return json.RawMessage(`{"list":[]}`), nil
}

func TestValidateRange(t *testing.T) {
	ok := [][2]string{{"2026-10-01", "2026-10-11"}, {"2025-11-01", "2025-12-01"}}
	for _, c := range ok {
		if _, _, err := ValidateRange(c[0], c[1], now); err != nil {
			t.Fatalf("%v: %v", c, err)
		}
	}
	bad := [][2]string{
		{"2026-10-01", "2026-10-12"}, // 今天
		{"2025-10-31", "2025-11-05"}, // 早于数据起始日
		{"2026-10-05", "2026-10-01"}, // 倒序
		{"2026-09-01", "2026-10-02"}, // 32 天
		{"2026/10/01", "2026-10-02"},
	}
	for _, c := range bad {
		if _, _, err := ValidateRange(c[0], c[1], now); err == nil {
			t.Fatalf("%v: want error", c)
		}
	}
}

func TestArticles(t *testing.T) {
	// 10-01 发表一篇推送文章（已满 7 天）；10-10 发表一篇未通知文章（未满 7 天，freepublish 有当天时刻）。
	f := &fakeCube{resp: map[string]string{
		"getarticletotaldetail 2026-10-01": `{"list":[{"ref_date":"2026-10-01","msgid":"100_1","publish_type":0,"title":"A","content_url":"http://mp.weixin.qq.com/s?mid=100&idx=1",
			"detail_list":[` + detail("2026-10-01", 300, 40, 0.01) + `,` + detail("2026-10-07", 500, 41, 0.01025) + `,` + detail("2026-10-11", 600, 42, 0.0105) + `]}],"is_delay":false}`,
		"getarticletotaldetail 2026-10-10": `{"list":[{"ref_date":"2026-10-10","msgid":"200_1","publish_type":1,"title":"B","content_url":"http://mp.weixin.qq.com/s?mid=200&amp;idx=1",
			"detail_list":[{"stat_date":"2026-10-10","read_user":10,"read_user_source":[],"read_delivery_rate":0}]}],"is_delay":true}`,
	}}
	b := &Builder{
		DataCube: f.DataCube,
		ListPublished: func(offset, count int) (json.RawMessage, error) {
			// 100_1 的 update_time 是后来编辑的时间（10-11），不能当发布时刻；200_1 是发表当天。
			return json.RawMessage(`{"total_count":2,"item":[
				{"update_time":` + fmt.Sprint(time.Date(2026, 10, 11, 0, 20, 0, 0, CST).Unix()) + `,"content":{"news_item":[{"url":"http://mp.weixin.qq.com/s?__biz=x&mid=100&idx=1&sn=y"}]}},
				{"update_time":` + fmt.Sprint(time.Date(2026, 10, 10, 21, 5, 0, 0, CST).Unix()) + `,"content":{"news_item":[{"url":"http://mp.weixin.qq.com/s?__biz=x&mid=200&idx=1&sn=z"}]}}]}`), nil
		},
		Now: now,
	}
	rep, err := b.Articles(day("2026-10-01"), day("2026-10-11"), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Articles) != 2 || !rep.IsDelay || len(f.calls) != 11 {
		t.Fatalf("articles=%d is_delay=%v calls=%d", len(rep.Articles), rep.IsDelay, len(f.calls))
	}
	a := rep.Articles[0]
	if a.MsgID != "100_1" || !a.Notified || a.Day0 == nil || a.Day0.ReadUser != 300 || a.Day0.PushReadUser != 40 {
		t.Fatalf("day0: %+v", a)
	}
	if a.Day7 == nil || a.Day7.StatDate != "2026-10-07" || a.Latest.StatDate != "2026-10-11" {
		t.Fatalf("day7/latest: %+v %+v", a.Day7, a.Latest)
	}
	if a.DeliveredEst == nil || *a.DeliveredEst != 4000 || *a.PushOpenRate != 0.0105 {
		t.Fatalf("delivered_est=%v rate=%v", a.DeliveredEst, a.PushOpenRate)
	}
	if _, ok := a.Latest.ReadBySource["全部"]; ok || a.Latest.ReadBySource["推荐"] != 558 {
		t.Fatalf("sources: %v", a.Latest.ReadBySource)
	}
	if a.PublishTime != nil {
		t.Fatalf("edited update_time must not be used: %v", *a.PublishTime)
	}
	if a.Series != nil {
		t.Fatal("series should be omitted")
	}
	c := rep.Articles[1]
	if c.Notified || c.DeliveredEst != nil || c.PushOpenRate != nil || c.Day7 != nil {
		t.Fatalf("unnotified article: %+v", c)
	}
	if c.PublishTime == nil || *c.PublishTime != "2026-10-10 21:05" || *c.PublishTimeSource != "freepublish" {
		t.Fatalf("publish time: %v %v", c.PublishTime, c.PublishTimeSource)
	}

	// 人工补录优先，且全部已补录时不再翻 freepublish。
	b.Meta = []store.ArticleMeta{{MsgID: "100_1", PublishedAt: "2026-10-01 21:00", Category: "攻略"}, {MsgID: "200_1", PublishedAt: "2026-10-10 08:00"}}
	b.ListPublished = func(int, int) (json.RawMessage, error) { t.Fatal("freepublish should not be called"); return nil, nil }
	rep, _ = b.Articles(day("2026-10-01"), day("2026-10-11"), true)
	a = rep.Articles[0]
	if *a.PublishTime != "2026-10-01 21:00" || *a.PublishTimeSource != "manual" || a.Category != "攻略" || len(a.Series) != 3 {
		t.Fatalf("manual meta: %+v", a)
	}
}

func TestArticlesPartialAndTotalFailure(t *testing.T) {
	e := errors.New("DataCube.getarticletotaldetail Error , errcode=45009 , errmsg=reach max api daily quota limit")
	f := &fakeCube{errs: map[string]error{"getarticletotaldetail 2026-10-02": e}}
	b := &Builder{DataCube: f.DataCube, Now: now}
	rep, err := b.Articles(day("2026-10-01"), day("2026-10-03"), false)
	if err != nil || len(rep.Warnings) != 1 || rep.Warnings[0].WechatErrcode != 45009 || rep.Warnings[0].Date != "2026-10-02" {
		t.Fatalf("partial: err=%v warnings=%+v", err, rep.Warnings)
	}
	if _, err := b.Articles(day("2026-10-02"), day("2026-10-02"), false); err != e {
		t.Fatalf("total failure should return the error, got %v", err)
	}
}

type memCache map[string]struct {
	body []byte
	at   time.Time
}

func (m memCache) Get(api, date string) ([]byte, time.Time, bool) {
	v, ok := m[api+date]
	return v.body, v.at, ok
}
func (m memCache) Put(api, date string, body []byte, at time.Time) {
	m[api+date] = struct {
		body []byte
		at   time.Time
	}{body, at}
}

func TestTotalDetailCache(t *testing.T) {
	f := &fakeCube{}
	c := memCache{}
	b := &Builder{DataCube: f.DataCube, Cache: c, Now: now}
	old, recent := day("2026-09-01"), day("2026-10-05") // 09-01 已过统计期 → 永久缓存；10-05 未定稿
	for i := 0; i < 2; i++ {
		_, _ = b.totalDetail(old)
		_, _ = b.totalDetail(recent)
	}
	if len(f.calls) != 2 {
		t.Fatalf("within TTL both should hit cache: %v", f.calls)
	}
	b.Now = now.Add(2 * time.Hour)
	_, _ = b.totalDetail(old)
	_, _ = b.totalDetail(recent)
	if len(f.calls) != 3 || !strings.Contains(f.calls[2], "2026-10-05") {
		t.Fatalf("after TTL only the unfinalized day should refetch: %v", f.calls)
	}
}

func TestDaily(t *testing.T) {
	// 10-02 推送、10-03 推送次日、10-04 不发；10-01 是额外取的前一天。
	f := &fakeCube{resp: map[string]string{
		"getarticletotaldetail 2026-10-02": `{"list":[{"ref_date":"2026-10-02","msgid":"1_1","publish_type":0},{"ref_date":"2026-10-02","msgid":"1_2","publish_type":0},{"ref_date":"2026-10-02","msgid":"9_1","publish_type":1}]}`,
		"getusersummary 2026-10-02":        `{"list":[{"ref_date":"2026-10-02","user_source":0,"new_user":5,"cancel_user":4},{"ref_date":"2026-10-02","user_source":57,"new_user":20,"cancel_user":0},{"ref_date":"2026-10-03","user_source":0,"new_user":3,"cancel_user":2}]}`,
		"getusercumulate 2026-10-01":       `{"list":[{"ref_date":"2026-10-01","cumulate_user":1000},{"ref_date":"2026-10-02","cumulate_user":1021},{"ref_date":"2026-10-03","cumulate_user":1022},{"ref_date":"2026-10-04","cumulate_user":1022}]}`,
		"getbizsummary 2026-10-02":         `{"list":[{"ref_date":"2026-10-02","detail":{"read_user":900,"read_user_source":[{"user_count":900,"scene_desc":"全部"},{"user_count":300,"scene_desc":"公众号消息"}],"share_user":12,"collection_user":4,"send_page_count":3}}],"is_delay":false}`,
	}}
	b := &Builder{DataCube: f.DataCube, Now: now}
	rep, err := b.Daily(day("2026-10-02"), day("2026-10-05"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Days) != 4 {
		t.Fatalf("days=%d", len(rep.Days))
	}
	d := rep.Days[0]
	if d.DayType != "pushed" || d.PushedCount != 2 || d.PublishedCount != 3 || *d.NewUser != 25 || *d.CancelUser != 4 || *d.NetUser != 21 {
		t.Fatalf("10-02: %+v", d)
	}
	if *d.CancelRate != 0.004 || d.NewBySource["57"] != 20 || *d.ReadUser != 900 || d.ReadBySource["公众号消息"] != 300 || *d.SendPageCount != 3 {
		t.Fatalf("10-02 rates/sources: %+v", d)
	}
	if rep.Days[1].DayType != "day_after_push" || rep.Days[2].DayType != "quiet" {
		t.Fatalf("day types: %s %s", rep.Days[1].DayType, rep.Days[2].DayType)
	}
	// 10-04 有累计、无增减记录 → 0；10-05 两者都没有 → null + warning。
	if *rep.Days[2].NewUser != 0 || *rep.Days[2].CancelRate != 0 || rep.Days[3].NewUser != nil {
		t.Fatalf("missing data handling: %+v %+v", rep.Days[2], rep.Days[3])
	}
	if len(rep.Warnings) != 1 || rep.Warnings[0].Date != "2026-10-05" {
		t.Fatalf("warnings: %+v", rep.Warnings)
	}
	if s := rep.Summary[0]; s.DayType != "pushed" || s.Days != 1 || *s.AvgNewUser != 25 {
		t.Fatalf("summary: %+v", rep.Summary)
	}
	// 取数：前 6 天 + 4 天的单篇明细（算近 7 天节奏），增减 1 段，累计从前一天起 1 段，汇总 1 段。
	if len(f.calls) != 13 {
		t.Fatalf("calls: %v", f.calls)
	}
	// 近 7 天只有 10-02 推送 → sparse；整段节奏相同，合成一个区间。
	if p := rep.Days[0].PushDays7d; p == nil || *p != 1 || rep.Days[0].Cadence != "sparse" {
		t.Fatalf("cadence: %+v", rep.Days[0])
	}
	if len(rep.CadencePeriods) != 1 || rep.CadencePeriods[0].Begin != "2026-10-02" || rep.CadencePeriods[0].Days != 4 {
		t.Fatalf("periods: %+v", rep.CadencePeriods)
	}
	if c := rep.ByCadence[2]; c.Cadence != "sparse" || c.Days != 4 || *c.AvgNewUser != 28.0/3 {
		t.Fatalf("by cadence: %+v", rep.ByCadence)
	}

	// 再查一次：定稿日期全部命中缓存（memCache），只有未定稿的日期重新调用。
	c := memCache{}
	b.Cache = c
	f.calls = nil
	_, _ = b.Daily(day("2026-10-02"), day("2026-10-05"))
	first := len(f.calls)
	b.Now = now.Add(2 * time.Hour)
	f.calls = nil
	_, _ = b.Daily(day("2026-10-02"), day("2026-10-05"))
	// 增减 / 累计 / 汇总的日期都已定稿（≤ 10-10）→ 全部命中；单篇明细仍在 30 天统计期内 → 过期重取。
	for _, call := range f.calls {
		if !strings.HasPrefix(call, apiTotalDetail) {
			t.Fatalf("final range dates should be cached, got %v", f.calls)
		}
	}
	if first != 13 || len(f.calls) != 10 {
		t.Fatalf("first=%d second=%v", first, f.calls)
	}
}

func TestCadenceAndRuns(t *testing.T) {
	for n, want := range map[int]string{0: "none", 1: "sparse", 2: "sparse", 3: "every_other_day", 5: "every_other_day", 6: "daily", 7: "daily"} {
		if got := cadenceOf(n); got != want {
			t.Fatalf("cadenceOf(%d)=%s want %s", n, got, want)
		}
	}
	ds := []time.Time{day("2026-10-01"), day("2026-10-02"), day("2026-10-03"), day("2026-10-05"), day("2026-10-06")}
	got := runs(ds, 2)
	want := [][2]string{{"2026-10-01", "2026-10-02"}, {"2026-10-03", "2026-10-03"}, {"2026-10-05", "2026-10-06"}}
	if len(got) != len(want) {
		t.Fatalf("runs: %v", got)
	}
	for i := range want {
		if fmtDate(got[i][0]) != want[i][0] || fmtDate(got[i][1]) != want[i][1] {
			t.Fatalf("runs[%d]=%v want %v", i, got[i], want[i])
		}
	}
}

func TestCachedRangeSplitsAndRefetchesMissing(t *testing.T) {
	f := &fakeCube{resp: map[string]string{
		"getusersummary 2026-10-01": `{"list":[{"ref_date":"2026-10-01","new_user":1},{"ref_date":"2026-10-03","new_user":3}]}`,
		"getusersummary 2026-10-11": `{"list":[]}`,
	}}
	c := memCache{}
	b := &Builder{DataCube: f.DataCube, Cache: c, Now: now}
	noFail := func(string, string, error) { t.Fatal("unexpected failure") }
	out, n := b.cachedRange(apiUserSummary, day("2026-10-01"), day("2026-10-03"), 7, noFail)
	if n != 1 || len(out) != 3 || !strings.Contains(string(out["2026-10-03"]), `"new_user":3`) || strings.Contains(string(out["2026-10-01"]), "10-03") {
		t.Fatalf("split: n=%d out=%v", n, out)
	}
	// 10-02 已缓存且定稿；扩展到 10-01–10-11，只有 10-04–10-11 缺失 → 一段调用（不超过 7 天则分两段）。
	f.calls = nil
	_, n = b.cachedRange(apiUserSummary, day("2026-10-01"), day("2026-10-11"), 7, noFail)
	if n != 2 || f.calls[0] != "getusersummary 2026-10-04 2026-10-10" || f.calls[1] != "getusersummary 2026-10-11 2026-10-11" {
		t.Fatalf("refetch: n=%d calls=%v", n, f.calls)
	}
}

func TestGroupArticles(t *testing.T) {
	snap := func(read, push int64) *types.ArticleSnapshot {
		return &types.ArticleSnapshot{ReadUser: read, PushReadUser: push}
	}
	rate := func(v float64) *float64 { return &v }
	tm := func(s string) *string { return &s }
	arts := []types.ArticleStats{
		{MsgID: "1_1", Category: "攻略", Notified: true, Day7: snap(100, 10), PushOpenRate: rate(0.02), PublishTime: tm("2026-10-01 21:00")},
		{MsgID: "2_1", Category: "攻略", Notified: true, Day7: snap(300, 30), PushOpenRate: rate(0.04), PublishTime: tm("2026-10-02 21:30")},
		{MsgID: "3_1", Category: "攻略", Notified: true, Day7: snap(1000, 50), PushOpenRate: rate(0.03), PublishTime: tm("2026-10-03 08:00")},
		{MsgID: "4_1", Notified: true, PushOpenRate: rate(0.01)},                               // 未分类、未满 7 天、无推送时刻
		{MsgID: "5_1", Category: "快讯", Day7: snap(50, 0), PublishTime: tm("2026-10-04 12:00")}, // 未通知：不进推送小时
	}
	byCat, byHour, missing := groupArticles(arts)
	if len(byCat) != 3 || byCat[0].Key != "快讯" || byCat[1].Key != "攻略" || byCat[2].Key != "未分类" {
		t.Fatalf("category order: %+v", byCat)
	}
	g := byCat[1]
	if g.Articles != 3 || g.Day7Articles != 3 || *g.AvgDay7Read != 1400.0/3 || *g.MedianDay7Read != 300 || *g.MedianPushOpenRate != 0.03 {
		t.Fatalf("攻略: %+v", g)
	}
	if u := byCat[2]; u.Day7Articles != 0 || u.AvgDay7Read != nil || *u.AvgPushOpenRate != 0.01 {
		t.Fatalf("未分类: %+v", u)
	}
	if missing != 1 || len(byHour) != 2 || byHour[0].Key != "08" || byHour[1].Key != "21" || byHour[1].Articles != 2 || *byHour[1].AvgPushOpenRate != 0.03 {
		t.Fatalf("by hour: missing=%d %+v", missing, byHour)
	}
}

func TestDailyAllFail(t *testing.T) {
	e := errors.New("errcode=48001")
	b := &Builder{DataCube: func(string, string, string) (json.RawMessage, error) { return nil, e }, Now: now}
	if _, err := b.Daily(day("2026-10-02"), day("2026-10-03")); err != e {
		t.Fatalf("want error, got %v", err)
	}
}

func TestArchive(t *testing.T) {
	f := &fakeCube{}
	c := memCache{}
	b := &Builder{DataCube: f.DataCube, Cache: c, Now: now}
	// MinDate(2025-11-01)–昨天(2026-10-11) 共 345 天：增减 / 累计各 50 段，汇总 12 段，单篇明细 345 天。
	calls, failed, stopped := b.Archive(100)
	if calls != 100 || failed != 0 || stopped != errBudget.Error() {
		t.Fatalf("first run: calls=%d failed=%d stopped=%q", calls, failed, stopped)
	}
	calls, _, stopped = b.Archive(1000)
	if stopped != "" || calls != 50+50+12+345-100 {
		t.Fatalf("second run should finish the backlog: calls=%d stopped=%q", calls, stopped)
	}
	// 第三次：定稿数据都在缓存，未定稿的也还在 1 小时有效期内 → 不调用。
	if calls, _, _ = b.Archive(1000); calls != 0 {
		t.Fatalf("third run within TTL should not call: %d", calls)
	}
	// 一小时后：未定稿部分重取 —— 区间接口只有昨天（各 1 段），单篇明细为发表未满统计期的 09-12–10-11（30 天）。
	b.Now = now.Add(2 * time.Hour)
	if calls, _, _ = b.Archive(1000); calls != 3+30 {
		t.Fatalf("refresh run: calls=%d", calls)
	}
}

func TestArchiveStopsOnUnauthorized(t *testing.T) {
	n := 0
	b := &Builder{DataCube: func(string, string, string) (json.RawMessage, error) {
		n++
		return nil, errors.New("DataCube Error , errcode=48001 , errmsg=api unauthorized")
	}, Cache: memCache{}, Now: now}
	calls, failed, stopped := b.Archive(1000)
	if n != 1 || calls != 1 || failed != 1 || !strings.Contains(stopped, "48001") {
		t.Fatalf("n=%d calls=%d failed=%d stopped=%q", n, calls, failed, stopped)
	}
}
