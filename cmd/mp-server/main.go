package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/silenceper/wechat/v2/cache"
	"github.com/silenceper/wechat/v2/util"

	"github.com/azure1489/mp-helper/internal/api"
	"github.com/azure1489/mp-helper/internal/config"
	"github.com/azure1489/mp-helper/internal/store"
	"github.com/azure1489/mp-helper/internal/wechat"
)

func main() {
	cfgPath := flag.String("config", os.Getenv("MP_HELPER_CONFIG"), "path to YAML config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	// silenceper 的 util 用包级 DefaultHTTPClient，启动时设置全局超时（仅此一次，serve 前完成，无并发问题）。
	if cfg.WechatTimeoutSeconds > 0 {
		util.DefaultHTTPClient.Timeout = time.Duration(cfg.WechatTimeoutSeconds) * time.Second
	}

	mgr := wechat.NewManager(cache.NewMemory())
	srv := api.NewServer(cfg, st, mgr)

	log.Printf("mp-server listening on %s (db=%s)", cfg.Listen, cfg.DBPath)
	if err := http.ListenAndServe(cfg.Listen, srv.Router()); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
