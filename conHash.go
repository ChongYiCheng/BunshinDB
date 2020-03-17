package main

import (
	"crypto/md5"
	"fmt"
	"math"
)

func toChar(i int) rune {
	return rune('A' - 1 + i)
}


func main() {
	//Set constants here
	const NUMBER_OF_VNODES = 3;
	const KEYSPACE = 20

	r := newRing(KEYSPACE)
	n := newNode(1, NUMBER_OF_VNODES)

	n.registerWithRing(r)

}

type ring struct {
	maxID int // 0 to maxID inclusive
	nodeMap map[int]node
	idArray []string
	nodeArray []node
}

type node struct {
//	keep it basic for now. need to integrate with Cheow Fu's later
	id string
	cName string //Canonical Name, like "A"
 	// Number of tokens to replicate
	numTokens int
	nodeAddresses []int
	nodeDataArray []nodeData
}

// A simple struct to define the schema when sending the payload containing basic data
type nodeData struct {
	id string
	hash int
	physicalNode node
}



// Constructor functions
func newNodeData(id string, hash int, physicalNode node) nodeData{
	return nodeData{id, hash,physicalNode}
}

func newNode(numID int, numTokens int) node{
	return node{string(toChar(numID)) + "0", string(toChar(numID)),
		numTokens, []int {}, []nodeData{}}
}

func newRing(maxID int) ring {
	return ring{maxID, make(map[int]node), []string {}, []node {}}
}

func (n node) registerWithRing(r ring) {
	nodeAddresses := []int {}
	//TODO: Can we do deduplication on the node side?
	for i := 0; i < n.numTokens +1; i ++ {
		id := fmt.Sprintf("%s%d", n.cName, i)
		hash := hashMD5(id, i, 0, r.maxID)
		nodeAddresses = append(nodeAddresses, hash)
		n.nodeDataArray = append(n.nodeDataArray, newNodeData(id, hash, n))
		//fmt.Println(fmt.Sprintf("%s%d", n.cName, i), n)
	}
	fmt.Printf("Node %s registering %s \n", n.id, n.toString(n.nodeDataArray))
	n.nodeDataArray = r.registerNodes(n.nodeDataArray)
	fmt.Printf("Ring registered for %s: %s  \n", n.id, n.toString(n.nodeDataArray))
}


func (r ring) registerNodes(nodeDataArray []nodeData) []nodeData{
	for _, nd := range nodeDataArray {
		for {
			if _, ok := r.nodeMap[nd.hash]; ok {
				nd.hash +=1
			} else {
				r.nodeMap[nd.hash] = nd.physicalNode
				break
			}
		}
	}
	return nodeDataArray
}

//Easy toString method

func (n node) toString(nodeDataArray []nodeData) []string{
	ret := []string {}
	for _, nd := range nodeDataArray {
		ret = append(ret, fmt.Sprintf("(Node %s, hash: %d) ", nd.id, nd.hash))
	}
	return ret
}


//write a method to generate 4 keys given a single node
//TODO: need to improve this further
func hashMD5(text string, salt int, min int, max int) int {

	hash:=md5.New()
	byteArray := hash.Sum([]byte(text))
	var output int
	for i, num := range byteArray{
		output += int(math.Pow(float64(num), float64(i % 5 + 2))) + salt * i
	}

	return output % (max - min + 1) + min
}

