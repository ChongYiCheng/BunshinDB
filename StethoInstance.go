package main

import (
	"50.041-DistSysProject-BunshinDB/Stetho"
	"fmt"
)

func main() {
	fmt.Println("Stetho running...")
	s:= Stetho.NewStetho("5000", 1, 5)
	s.Start()
}
