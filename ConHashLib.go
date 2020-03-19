package main

import (
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/dgraph-io/badger"
)

type Node struct{
	id string
	cName string
	numTokens int
	//quitChannel chan struct{}
	//nodeChannel chan interface{}
	DBPath string // e.g /tmp/badger
	nodeDB *badger.DB
	ip string //a.b.c.d:port
	port string
	//allNodes map[int]string
	//localClock []int

	nodeRingPositions []int
	ring *Ring

	//{name: str, nodeRingPosition: int}
	nodeDataArray []nodeData
}

type Ring struct{
	maxID int // 0 to maxID inclusive
	ringNodeDataArray []nodeData
}

type nodeData struct{
	//Node Data contains ip and port and hash This helps in determining which node is responsible for
	//Which request(read/write) and contains relevant info (ip:port) to
	//direct data to that node
	id string
	cName string
	hash int
	ip string
	port string
}


func toChar(i int) rune {
	return rune('A' - 1 + i)
}


func newNodeData(id string, cName string, hash int,ip string, port string) nodeData{
	return nodeData{id, cName, hash, ip, port}
}

func newNode(numID int, numTokens int, DBPath string, ip string, port string, ring *Ring) *Node{
	return &Node{
		id:string(toChar(numID)) + "0",
		cName:string(toChar(numID)),
		numTokens: numTokens,
		DBPath: DBPath,
		ip: ip,
		port: port,
		ring: ring,
	}
}

func newRing(maxID int) *Ring{
	nodeDataArray := make([]nodeData, maxID, maxID)
	fmt.Println(len(nodeDataArray))
	fmt.Println(nodeDataArray[1].id)
	return &Ring{maxID, nodeDataArray}
}

//node will create numTokens worth of virtual nodes
func (n *Node) registerWithRing(r *Ring) {
	nodeDataArray := []nodeData {}
	//copy(tempNodeDataArray,localRing.ringNodeDataArray)
	//TODO: Can we do deduplication on the node side?
	for i := 0; i < n.numTokens +1; i ++ {
		id := fmt.Sprintf("%s%d", n.cName, i)
		hash := hashMD5(id, 0, r.maxID)
		nodeDataArray = append(nodeDataArray, newNodeData(id, n.cName, hash, n.ip, n.port))
	}

	fmt.Printf("Node %s registering %s \n", n.id, toString(nodeDataArray))
	n.nodeDataArray = r.registerNodes(nodeDataArray)
	fmt.Printf("Ring registered for %s: %s  \n", n.id, toString(n.nodeDataArray))
}

func (r *Ring) registerNodes(nodeDataArray []nodeData) []nodeData{
	ret := []nodeData{}
	for _, nd := range nodeDataArray {
		for {
			//if occupied, we do linear probing
			if r.ringNodeDataArray[nd.hash].id != "" {
				nd.hash = (nd.hash + 1) % len(nodeDataArray)
			} else {
				r.ringNodeDataArray[nd.hash] = nd
				ret = append(ret, nd)
				break
			}

		}
	}
	return ret
}

//func toString() string {
//
//}
//Easy toString method
func toString(nodeDataArray []nodeData) []string{
	ret := []string {}
	for _, nd := range nodeDataArray {
		ret = append(ret, fmt.Sprintf("(%s, %d) ", nd.id, nd.hash))
	}
	return ret
}

//string must end with an integer
func (r *Ring) getNode(id string) (string, error) {
	var NodeNotFound = errors.New("Node not found")
	hash := hashMD5(id, 0, r.maxID)

	//Impose an upper bound for probe times
	for i:= 0; i < len(r.ringNodeDataArray); i ++{
		fmt.Println(r.ringNodeDataArray[hash].id)
		if r.ringNodeDataArray[hash].id == id {
			ip_port := fmt.Sprintf("%s:%s",r.ringNodeDataArray[hash].ip,r.ringNodeDataArray[hash].port)
			//return r.nodeArray[hash].physicalNode, nil
			return ip_port, nil
		}
		hash = (hash + 1) % len(r.ringNodeDataArray)
	}

	return id, NodeNotFound
}

func hashMD5(text string, min int, max int) int {
	byteArray := md5.Sum([] byte(text))
	var output int
	for _, num := range byteArray{
		output += int(num)
	}

	return output % (max - min + 1) + min
}
