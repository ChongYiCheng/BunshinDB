package main

//merkleTree Playground code
import (
	"bytes"
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
	discrepancies := compareTwoTrees(mr1, mr2)

	fmt.Println(discrepancies)

	//Get the Merkle Root of the tree
	//mr := t.MerkleRoot()
	//log.Println(mr)
	//
	////Verify the entire tree (hashes for each node) is valid
	//vt, err := t.VerifyTree()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//log.Println("Verify Tree: ", vt)
	//
	////Verify a specific content in in the tree
	//vc, err := t.VerifyContent(list1[0])
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//
	//log.Println("Verify Content: ", vc)
	//
	////String representation
	//log.Println(t)
}

func compareTwoTrees(root1 *merkletree.Node, root2 *merkletree.Node) [][]merkletree.Content {
	discrepancies := [][]merkletree.Content{}
	compareTwoNodes(root1, root2, &discrepancies)

	return discrepancies
}

func compareTwoNodes(node1 *merkletree.Node, node2 *merkletree.Node, discrepancies *[][]merkletree.Content) {
	fmt.Println(node1, node2)
	if node1.IsLeaf() || node2.IsLeaf() {
		fmt.Println("CONTENTTTT ", node1.C, node2.C)
		if bytes.Compare(node1.Hash, node2.Hash) != 0 {
			newEntry := []merkletree.Content{node1.C, node2.C}
			*discrepancies = append(*discrepancies, newEntry)
		}
	}else if bytes.Compare(node1.Hash, node2.Hash) == 0 {
		fmt.Println("Equal")
		return
	} else {
		fmt.Println("Not Equal")
		compareTwoNodes(node1.Left, node2.Left, discrepancies)
		compareTwoNodes(node1.Right, node2.Right, discrepancies)
	}
}