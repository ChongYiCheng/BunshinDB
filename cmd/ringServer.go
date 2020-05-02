package main

import (
	"50.041-DistSysProject-BunshinDB/pkg/ConHash"
	"50.041-DistSysProject-BunshinDB/pkg/ServerUtils"
	"50.041-DistSysProject-BunshinDB/config"
)

func main() {

	const MAX_KEY = config.MAX_KEY
	const REPLICATION_FACTOR = config.REPLICATION_FACTOR
	const RW_FACTOR = config.RW_FACTOR
	const REGISTER_ENDPOINT = config.RING_REGISTER_ENDPOINT
	const STETHO_URL = config.STETHO_URL
	const STETHO_SERVER_PORT = config.STETHO_SERVER_PORT
	const RING_SERVER_PORT = config.RING_SERVER_PORT

	ring := ConHash.NewRing(MAX_KEY,REPLICATION_FACTOR,RW_FACTOR)


	ringServer := ServerUtils.NewRingServer(*ring, STETHO_URL, RING_SERVER_PORT)
	ringServer.RegisterWithStetho(REGISTER_ENDPOINT)
	ringServer.Start()
}
