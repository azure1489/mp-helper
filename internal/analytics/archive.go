package analytics

import (
	"encoding/json"
	"errors"
	"strconv"
)

// errBudget 表示本次存档的微信调用次数已用完。
var errBudget = errors.New("call budget exhausted")

// 遇到这些错误码立即停止该账号的存档：48001 无接口权限，45009 当日调用次数用完。
var stopCodes = map[int64]bool{48001: true, 45009: true}

// Archive 把账号从 MinDate 到昨天缺失或未定稿的统计补进缓存，最多调用微信 budget 次。
// 先补增减、累计、汇总（按区间调用，次数少），再从昨天往前补单篇明细。
// 返回实际调用次数、失败次数与提前停止的原因（为空表示全部补齐）。
func (b *Builder) Archive(budget int) (calls, failed int, stopped string) {
	if b.Cache == nil {
		return 0, 0, "no cache"
	}
	var stop error
	inner := *b
	inner.DataCube = func(api, begin, end string) (json.RawMessage, error) {
		if stop != nil {
			return nil, stop
		}
		if calls >= budget {
			stop = errBudget
			return nil, stop
		}
		calls++
		raw, err := b.DataCube(api, begin, end)
		if err != nil {
			failed++
			if m := errcodeRe.FindStringSubmatch(err.Error()); len(m) == 2 {
				if code, _ := strconv.ParseInt(m[1], 10, 64); stopCodes[code] {
					stop = err
				}
			}
		}
		return raw, err
	}
	ignore := func(string, string, error) {}
	yesterday := today(b.Now).AddDate(0, 0, -1)
	inner.cachedRange(apiUserSummary, MinDate, yesterday, 7, ignore)
	inner.cachedRange(apiUserCumul, MinDate, yesterday, 7, ignore)
	inner.cachedRange(apiBizSummary, MinDate, yesterday, 30, ignore)
	for d := yesterday; !d.Before(MinDate) && stop == nil; d = d.AddDate(0, 0, -1) {
		_, _ = inner.totalDetail(d)
	}
	if stop != nil {
		stopped = stop.Error()
	}
	return calls, failed, stopped
}
