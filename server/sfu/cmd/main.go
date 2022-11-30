package main

import (
	"server/server/sfu/src"
)

func close() {
	src.Stop()
}

func main() {
	defer close()
	src.Start()
	select {}
}
