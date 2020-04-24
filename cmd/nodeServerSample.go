package main

import (
	"50.041-DistSysProject-BunshinDB/pkg/ConHash"
	"50.041-DistSysProject-BunshinDB/pkg/ConHttp"
	"log"
)

func main() {

	const RING_URL = "http://10.12.7.122:5001"

	ip, err := ConHttp.ExternalIP()
	if err != nil {
		log.Fatalln(err)
	}
	node := ConHash.NewNodeServer(1, 4, "/tmp/Badger8080", ip, "8002")
	node.RegisterWithRingServer(RING_URL)
}
