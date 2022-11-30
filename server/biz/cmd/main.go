package main

import (
	"server/server/biz/src"
)

func close() {
	src.Stop()
}

func main() {
	defer close()
	src.Start()
	select {}
}
