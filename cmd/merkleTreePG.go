package main

//merkleTree Playground code
import (
	"crypto/sha256"
	"fmt"
	"log"
	"github.com/lionellloh/merkletree"
)

//TestContent implements the Content interface provided by merkletree and represents the content stored in the tree.
type TestContent struct {
	x string
}

//CalculateHash hashes the values of a TestContent
func (t TestContent) CalculateHash() ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(t.x)); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

//Equals tests for equality of two Contents
func (t TestContent) Equals(other merkletree.Content) (bool, error) {
	return t.x == other.(TestContent).x, nil
}

func main() {
	//Build list1 of Content to build tree
	var list1 []merkletree.Content
	list1 = append(list1, TestContent{x: "Hello"})
	list1 = append(list1, TestContent{x: "Hi"})
	list1 = append(list1, TestContent{x: "Hey"})
	list1 = append(list1, TestContent{x: "Hola"})

	list2 := make([]merkletree.Content, len(list1))

	copy(list2, list1)
	list2[3] = TestContent{x:"Holb"}
	list2[0] = TestContent{x:"hel"}

	//Create a new Merkle Tree from the list1 of Content
	t1, err := merkletree.NewTree(list1)
	if err != nil {
		log.Fatal(err)
	}

	t2, err := merkletree.NewTree(list2)
	if err != nil {
		log.Fatal(err)
	}

	mr1 := t1.MerkleRootNode()
	mr2 := t2.MerkleRootNode()
	discrepancies := merkletree.CompareTwoTrees(mr1, mr2)

	fmt.Println(discrepancies)

}

