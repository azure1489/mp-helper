package api

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/azure1489/mp-helper/internal/analytics"
	"github.com/azure1489/mp-helper/internal/types"
)

// 每日统计存档：定时（或经管理接口手动）把各账号缺失或未定稿的统计补进本地库，
// 一来分析时直接读库、少耗微信调用次数，二来微信下线或调整接口时历史数据仍在。

type archiveState struct {
	mu       sync.Mutex
	running  bool
	started  time.Time
	finished time.Time
	results  []types.ArchiveAccountResult
}

// startArchive 若当前没有存档在跑，则在后台启动一次并返回 true。
func (s *Server) startArchive() bool {
	s.archive.mu.Lock()
	defer s.archive.mu.Unlock()
	if s.archive.running {
		return false
	}
	s.archive.running = true
	s.archive.started = time.Now()
	go s.runArchive()
	return true
}

func (s *Server) runArchive() {
	var results []types.ArchiveAccountResult
	defer func() {
		s.archive.mu.Lock()
		s.archive.running = false
		s.archive.finished = time.Now()
		s.archive.results = results
		s.archive.mu.Unlock()
	}()
	accs, err := s.store.ListAccounts()
	if err != nil {
		log.Printf("archive: list accounts: %v", err)
		return
	}
	for _, acc := range accs {
		b, err := s.builderFor(*acc)
		if err != nil {
			log.Printf("archive: account %d: %v", acc.ID, err)
			continue
		}
		calls, failed, stopped := b.Archive(s.cfg.Archive.MaxCalls)
		results = append(results, types.ArchiveAccountResult{AccountID: acc.ID, Name: acc.Name, Calls: calls, Failed: failed, Stopped: stopped})
		log.Printf("archive: account %d (%s): calls=%d failed=%d stopped=%q", acc.ID, acc.Name, calls, failed, stopped)
	}
}

// nextArchiveAt 返回 now 之后最近一次 HH:MM（北京时间）。
func nextArchiveAt(now time.Time, hhmm string) time.Time {
	t, _ := time.Parse("15:04", hhmm)
	n := now.In(analytics.CST)
	next := time.Date(n.Year(), n.Month(), n.Day(), t.Hour(), t.Minute(), 0, 0, analytics.CST)
	if !next.After(n) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// StartArchiver 按配置每天定时存档，直到 ctx 结束。
func (s *Server) StartArchiver(ctx context.Context) {
	for {
		next := nextArchiveAt(time.Now(), s.cfg.Archive.Time)
		log.Printf("archive: next run at %s", next.Format(time.RFC3339))
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(next)):
			if !s.startArchive() {
				log.Printf("archive: previous run still in progress, skipped")
			}
		}
	}
}

func (s *Server) handleRunArchive(c *gin.Context) {
	if !s.startArchive() {
		respondError(c, http.StatusConflict, "conflict", "an archive run is already in progress")
		return
	}
	c.JSON(http.StatusAccepted, s.archiveStatus())
}

func (s *Server) handleArchiveStatus(c *gin.Context) {
	c.JSON(http.StatusOK, s.archiveStatus())
}

func (s *Server) archiveStatus() types.ArchiveStatus {
	s.archive.mu.Lock()
	defer s.archive.mu.Unlock()
	st := types.ArchiveStatus{Running: s.archive.running, Results: s.archive.results}
	if st.Results == nil {
		st.Results = []types.ArchiveAccountResult{}
	}
	if !s.archive.started.IsZero() {
		st.LastStarted = s.archive.started.Format(time.RFC3339)
	}
	if !s.archive.finished.IsZero() {
		st.LastFinished = s.archive.finished.Format(time.RFC3339)
	}
	return st
}
