package main

import (
	"context"
	"go-drive/common"
	"go-drive/common/registry"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	ch := registry.NewComponentHolder()

	engine, e := Initialize(context.Background(), ch)
	if e != nil {
		log.Fatalln(e)
	}
	log.Println("Server started")

	log.Fatalln(http.ListenAndServe(ch.Get("config").(common.Config).Listen, engine))
}
