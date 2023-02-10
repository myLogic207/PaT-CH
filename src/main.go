package main

import (
	"time"

	"github.com/mylogic207/PaT-CH/cache"
)

func main() {
	Cache := cache.NewConnector("redis://default:redispw@localhost:49154", 0)
	Cache.Connect()
	defer Cache.Close()
	time.Sleep(10 * time.Second)
}
