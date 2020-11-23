package main

import (
	"go-drive/common"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	ch := common.NewComponentHolder()

	engine, e := Initialize(ch)
	if e != nil {
		log.Fatalln(e)
	}

	log.Fatalln(http.ListenAndServe(ch.Get("config").(common.Config).Listen, engine))
}
