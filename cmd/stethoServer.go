package main

import (
	"50.041-DistSysProject-BunshinDB/pkg/ServerUtils"
	"fmt"
	"50.041-DistSysProject-BunshinDB/config"
)

func main() {
	const PING_INTERVAL = config.PING_INTERVAL
	const TIMEOUT_INTERVAL = config.TIMEOUT_INTERVAL
	const FAILURE_THRESHOLD = config.FAILURE_THRESHOLD

	fmt.Println("Stetho running...")
	s:= ServerUtils.NewStethoServer("5000", PING_INTERVAL, TIMEOUT_INTERVAL, FAILURE_THRESHOLD)
	s.Start()
}
