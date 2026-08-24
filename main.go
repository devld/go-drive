package main

import (
	"context"
	"go-drive/common"
	"go-drive/common/logging"
	"go-drive/common/registry"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ch := registry.NewComponentHolder()

	engine, e := Initialize(context.Background(), ch)
	if e != nil {
		logging.For("start").Errorf("initialization failed: %v", e)
		os.Exit(1)
	}

	dispose := func() { _ = ch.Dispose() }

	conf := ch.Get(registry.KeyConfig).(common.Config)
	server := &http.Server{
		Addr:     conf.Listen,
		Handler:  engine,
		ErrorLog: logging.For("http-s").StdLogger(),
	}
	logging.For("server").Infof("starting HTTP server listen=%s", conf.Listen)

	go func() {
		if e := server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			dispose()
			logging.For("server").Errorf("listen failed: %v", e)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logging.For("server").Infof("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if e := server.Shutdown(ctx); e != nil {
		dispose()
		logging.For("server").Errorf("shutdown failed: %v", e)
		os.Exit(1)
	}

	dispose()
}
