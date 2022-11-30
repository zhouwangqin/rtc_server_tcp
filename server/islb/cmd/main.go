package main

import (
	"server/server/islb/src"
)

func close() {
	src.Stop()
}

func main() {
	defer close()
	src.Start()
	select {}
}
