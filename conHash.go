package main

import (
	"crypto/md5"
	"fmt"
	"math"
)

const NUMBER_OF_VNODES = 10;


func main() {
	r := newRing(15)
	n := newNode(1)

	n.registerWithRing(r)

}

type ring struct {
	maxID int // 0 to maxID inclusive
	nodeMap map[int]node

}

type node struct {
//	keep it basic for now. need to integrate with Cheow Fu's later
	id int
	// Number of vNodes to replicate
	numTokens int
	nodeAddresses []int
}

func newNode(id int) node{
	return node{id, NUMBER_OF_VNODES, []int {}}
}

func newRing(maxID int) ring {
	return ring{maxID, make(map[int]node)}
}

func (n node) registerWithRing(r ring) {
	nodeAddresses := []int {}
	for i := 0; i < n.numTokens +1; i ++ {
		id := hashMD5("Node" + string(n.id), i, 0, r.maxID)
		nodeAddresses = append(nodeAddresses, id)
	}
	fmt.Println("Node:", nodeAddresses)
	n.nodeAddresses = r.registerNodes(nodeAddresses, n)
}


func (r ring) registerNodes(idArray []int, n node) []int{
	ret := []int{}
	for _, id := range idArray {
		for {
			if _, ok := r.nodeMap[id]; ok {
				id += 1
			} else {
				r.nodeMap[id] = n
				ret = append(ret, id)
				break
			}
		}
	}
	fmt.Println("Ring:", ret)
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

