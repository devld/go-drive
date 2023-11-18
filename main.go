package main

import (
	"context"
	"go-drive/common"
	"go-drive/common/registry"
	"log"
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
		log.Fatalln(e)
	}

	dispose := func() { _ = ch.Dispose() }

	conf := ch.Get("config").(common.Config)
	server := &http.Server{Addr: conf.Listen, Handler: engine}

	go func() {
		if e := server.ListenAndServe(); e != nil && e != http.ErrServerClosed {
			dispose()
			log.Fatalln(e)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Signal received. Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if e := server.Shutdown(ctx); e != nil {
		dispose()
		log.Fatalln(e)
	}

	dispose()
}
