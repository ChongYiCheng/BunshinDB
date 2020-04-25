package main

import (
	"50.041-DistSysProject-BunshinDB/pkg/ConHash"
	"50.041-DistSysProject-BunshinDB/pkg/ServerUtils"
)

func main() {

	const MAX_KEY = 100;
	const REPLICATION_FACTOR = 3;
	const RW_FACTOR = 1;
	const REGISTER_ENDPOINT = "set-ring"
	const STETHO_URL = "http://192.168.1.142:5000"
	const RING_SERVER_PORT = "5001"
	const STEHO_SERVER_PORT = "5000"

	ring := ConHash.NewRing(MAX_KEY,REPLICATION_FACTOR,RW_FACTOR)

	ringServer := ServerUtils.NewRingServer(*ring, STETHO_URL, RING_SERVER_PORT)
	ringServer.RegisterWithStetho(STEHO_SERVER_PORT, REGISTER_ENDPOINT)
	ringServer.Start()
}
