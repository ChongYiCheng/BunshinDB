package main

import (
	"crypto/md5"
	"errors"
	"fmt"
)

func toChar(i int) rune {
	return rune('A' - 1 + i)
}


func main() {
	//Set constants here
	const NUMBER_OF_VNODES = 15;
	const MAX_KEY = 20

	r := newRing(MAX_KEY)
	n := newNode(1, NUMBER_OF_VNODES)
	n.registerWithRing(r)

	node, err := r.getNode("A2")
	if err == nil {
		fmt.Printf("Node found: %s \n", node.id)
	}
}

type ring struct {
	maxID int // 0 to maxID inclusive
	nodeArray []nodeData
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
	nodeDataArray := []nodeData {}
	nodeDataArray = make([]nodeData, maxID, maxID)
	fmt.Println(len(nodeDataArray))
	fmt.Println(nodeDataArray[1].id)
	return ring{maxID, nodeDataArray}
}

func (n node) registerWithRing(r ring) {
	nodeAddresses := []int {}
	//TODO: Can we do deduplication on the node side?
	for i := 0; i < n.numTokens +1; i ++ {
		id := fmt.Sprintf("%s%d", n.cName, i)
		hash := hashMD5(id, 0, r.maxID)
		nodeAddresses = append(nodeAddresses, hash)
		n.nodeDataArray = append(n.nodeDataArray, newNodeData(id, hash, n))
		//fmt.Println(fmt.Sprintf("%s%d", n.cName, i), n)
	}
	fmt.Printf("Node %s registering %s \n", n.id, toString(n.nodeDataArray))
	n.nodeDataArray = r.registerNodes(n.nodeDataArray)
	fmt.Printf("Ring registered for %s: %s  \n", n.id, toString(n.nodeDataArray))
}


func (r ring) registerNodes(nodeDataArray []nodeData) []nodeData{
	ret := []nodeData{}
	for _, nd := range nodeDataArray {
		for {
			//if occupied, we do linear probing
			if r.nodeArray[nd.hash].id != "" {
				nd.hash = (nd.hash + 1) % len(nodeDataArray)
			} else {
				r.nodeArray[nd.hash] = nd
				ret = append(ret, nd)
				break
			}

		}
	}
	return ret
}

//Easy toString method
func toString(nodeDataArray []nodeData) []string{
	ret := []string {}
	for _, nd := range nodeDataArray {
		ret = append(ret, fmt.Sprintf("(%s, %d) ", nd.id, nd.hash))
	}
	return ret
}

//string must end with an integer
func (r ring) getNode(id string) (node, error) {
	var NodeNotFound = errors.New("Node not found")
	hash := hashMD5(id, 0, r.maxID)

	//Impose an upper bound for probe times
	for i:= 0; i < len(r.nodeArray); i ++{
		fmt.Println(r.nodeArray[hash].id)
		if r.nodeArray[hash].id == id {
			return r.nodeArray[hash].physicalNode, nil
		}
		hash = (hash + 1) % len(r.nodeArray)
	}

	return node{}, NodeNotFound
}

//write a method to generate 4 keys given a single node
//TODO: need to improve this further
func hashMD5(text string, min int, max int) int {
	byteArray := md5.Sum([] byte(text))
	var output int
	for _, num := range byteArray{
		output += int(num)
	}

	return output % (max - min + 1) + min
}



