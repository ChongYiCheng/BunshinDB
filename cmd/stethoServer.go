package main

import (
	"50.041-DistSysProject-BunshinDB/pkg/ServerUtils"
	"fmt"
)

func main() {
	const PING_INTERVAL = 1
	const TIMEOUT_INTERVAL = 1
	const FAILURE_THRESHOLD = 1

	fmt.Println("Stetho running...")
	s:= ServerUtils.NewStethoServer("5000", PING_INTERVAL, TIMEOUT_INTERVAL, FAILURE_THRESHOLD)
	s.Start()
}
