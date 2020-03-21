package main

import (
	"fmt"
	"sort"
)


/* ## packet structure ## */
type pkt struct {
	data string
	clk map[string]int
}


/*
###########################
## Reconciliation main code
###########################
*/
func reconciliation(pkts ...pkt) []pkt {

	//variables
	to_remove := make(map[int]struct{})		//to store indexes of packets to remove; removing packets in order for syntactic reconciliation (see notes below test cases txt)

	fmt.Printf("Coordinator node receiving packets...")
	//for _,pkt := range pkts { fmt.Printf("Received packet! data: %v, clk: %v\n", pkt.data, pkt.clk)}
	//time.Sleep(time.Second * time.Duration(2))
	fmt.Println("All packets received, R value reached")

	//creating pairs from the list by taking i (p1) & i+1 (p2)
	for p1 := 0; p1 < len(pkts)-1; p1++ {
		for p2 := p1+1; p2 < len(pkts); p2++ {
			rm_idx := pair_test(p1,p2,pkts[p1],pkts[p2])
			switch rm_idx {
				case -1:
					fmt.Println("Cannot be syntactic-ally reconciled.")
				case p1,p2:
					//fmt.Println("Removing :",rm_idx)
					if _,ok := to_remove[rm_idx]; !ok {
						to_remove[rm_idx] = struct{}{}
					}
				default:	//for err handling
					fmt.Println("Error: rm_idx out of range")
			}
			fmt.Println(".")
		}
	}

	//convert to_remove to slice, so can iterate backwards
	rm := make([]int, 0, len(to_remove))
	for k := range to_remove { rm = append(rm, k)}
	sort.Ints(rm)
	//remove packets with indexes found in to_remove
	for j := range rm {
		i := rm[len(rm)-1-j]
		// Shift a[i+1:] left one index; copy a[i+1:] to a[i:]	
		copy(pkts[i:], pkts[i+1:])
		pkts = pkts[:len(pkts)-1]
	}

	fmt.Println("Final list :", pkts)
	return pkts
}


/* 
##############################
## test pairs of vector clocks
##############################
return value is the one to discard (eg if p2 is the more updated value, return p1)
note: if want to be done with goroutines, the return value can be put directly on to_remove
*/
func pair_test(p1 int, p2 int, pkt1 pkt, pkt2 pkt) int {

	fmt.Printf("Testing - %v : %v , %v : %v\n",pkt1.data,pkt1.clk,pkt2.data,pkt2.clk)

	//check if same number of vectors
	if len(pkt1.clk) != len(pkt2.clk) {
		fmt.Println("Clocks are of different lengths")
		return -1
	}

	//check if pair given has the same node(s)
	for k := range pkt1.clk {
		_,check := pkt2.clk[k]
		if !check {
			fmt.Println("Different nodes found")
			return -1
		}
	}

	//if same node(s), check if higher lower
	pnn := 0
	for n,c1 := range pkt1.clk {
		c2 := pkt2.clk[n]
		switch {
			default:
				pnn = c1 - c2
			case pnn < 0:
				//c1 < c2
				if c1-c2 > 0 {
					fmt.Println("Mixed clock values")
					return -1
				}
			case pnn > 0:
				//c1 > c2
				if c1-c2 < 0 {
					fmt.Println("Mixed clock values")
					return -1
				}
		}
	}

	fmt.Println("clocks can be syntactic-ally reconciled")
	if pnn >= 0 {
		fmt.Printf("choosing %v : %v\n", pkt1.data, pkt1.clk)
		return p2
	} else {
		fmt.Printf("choosing %v : %v\n", pkt2.data, pkt2.clk)
		return p1
	}

	fmt.Println("Error: pnn value not real")
  return -1
}

/*
#########
## MAIN
#########
*/
func main() {

	//two_vector_test()
	full_test()

}

/*
############
## Test code
############
*/

func full_test() {

	fmt.Println("================================")
	fmt.Println("==TEST 1==")
	clk := make(map[string]int)
	clk["A0"] = 1
	clk["B0"] = 1
	d1 := pkt{"d1", clk}

	clk = make(map[string]int)
	clk["A0"] = 2
	clk["B0"] = 1
	d2 := pkt{"d2", clk}

	clk = make(map[string]int)
	clk["D0"] = 2
	clk["B0"] = 2
	d3 := pkt{"d3", clk}

	fmt.Printf("%v : %v , %v : %v, %v : %v\n",d1.data,d1.clk,d2.data,d2.clk,d3.data,d3.clk)
	fmt.Println("---------------------")
	reconciliation(d1,d2,d3)
	fmt.Println("================================")

	fmt.Println("==TEST 2==")
	clk = make(map[string]int)
	clk["B0"] = 2
	clk["A0"] = 1
	d4 := pkt{"d4", clk}

	fmt.Printf("%v : %v , %v : %v, %v : %v\n",d1.data,d1.clk,d2.data,d2.clk,d4.data,d4.clk)
	fmt.Println("---------------------")
	reconciliation(d1,d2,d4)
	fmt.Println("================================")

	fmt.Println("==TEST 3==")
	clk = make(map[string]int)
	clk["A0"] = 2
	clk["B0"] = 1
	clk["C0"] = 2
	d5 := pkt{"d5", clk}

	clk = make(map[string]int)
	clk["A0"] = 3
	clk["B0"] = 1
	clk["C0"] = 1
	d6 := pkt{"d6", clk}

	clk = make(map[string]int)
	clk["D0"] = 1
	clk["A0"] = 5
	clk["B0"] = 2
	clk["C0"] = 3
	d7 := pkt{"d7", clk}

	clk = make(map[string]int)
	clk["A0"] = 2
	clk["B0"] = 2
	clk["C0"] = 2
	clk["D0"] = 1
	d8 := pkt{"d8", clk}

	clk = make(map[string]int)
	clk["D0"] = 6
	d9 := pkt{"d9", clk}

	clk = make(map[string]int)
	clk["C0"] = 1
	clk["A0"] = 1
	clk["B0"] = 1
	d10 := pkt{"d10", clk}

	fmt.Printf("%v : %v , %v : %v,\n", d1.data, d1.clk, d5.data, d5.clk)
	fmt.Printf("%v : %v , %v : %v,\n", d6.data, d6.clk, d7.data, d7.clk)
	fmt.Printf("%v : %v , %v : %v,\n", d3.data, d3.clk, d8.data, d8.clk)
	fmt.Printf("%v : %v , %v : %v, %v : %v\n", d9.data, d9.clk, d10.data, d10.clk, d2.data, d2.clk)
	fmt.Println("---------------------")

	reconciliation(d1,d5,d6,d7,d3,d8,d9,d10,d2)
	fmt.Println("================================")

}


func two_vector_test() {

	fmt.Println("### Simple pair tests ###")
	fmt.Println("================================")
	fmt.Println("==TEST 1==")
	clk := make(map[string]int)
	clk["A0"] = 1
	clk["B0"] = 1
	d1 := pkt{"d1", clk}

	clk = make(map[string]int)
	clk["A0"] = 2
	clk["B0"] = 1
	d2 := pkt{"d2", clk}

	fmt.Printf("%v : %v , %v : %v\n",d1.data,d1.clk,d2.data,d2.clk)
	reconciliation(d1,d2)
	fmt.Println("================================")

	fmt.Println("==TEST 2==")
	clk = make(map[string]int)
	clk["A0"] = 3
	clk["C0"] = 2
	d1 = pkt{"d1", clk}

	clk = make(map[string]int)
	clk["B0"] = 1
	clk["C0"] = 1
	d2 = pkt{"d2", clk}

	fmt.Printf("%v : %v , %v : %v\n",d1.data,d1.clk,d2.data,d2.clk)
	reconciliation(d1,d2)
	fmt.Println("================================")

	fmt.Println("==TEST 3==")
	clk = make(map[string]int)
	clk["D0"] = 1
	d1 = pkt{"d1", clk}

	clk = make(map[string]int)
	clk["D0"] = 2
	clk["A0"] = 1
	d2 = pkt{"d2", clk}

	fmt.Printf("%v : %v , %v : %v\n",d1.data,d1.clk,d2.data,d2.clk)
	reconciliation(d1,d2)
	fmt.Println("================================")

	fmt.Println("==TEST 4==")
	clk = make(map[string]int)
	clk["A0"] = 1
	clk["B0"] = 1
	d1 = pkt{"d1", clk}

	clk = make(map[string]int)
	clk["A0"] = 1
	clk["B0"] = 1
	d2 = pkt{"d2", clk}

	fmt.Printf("%v : %v , %v : %v\n",d1.data,d1.clk,d2.data,d2.clk)
	reconciliation(d1,d2)
	fmt.Println("================================")

	fmt.Println("==TEST 5==")
	clk = make(map[string]int)
	clk["A0"] = 2
	clk["B0"] = 2
	d1 = pkt{"d1", clk}

	clk = make(map[string]int)
	clk["A0"] = 1
	clk["B0"] = 1
	d2 = pkt{"d2", clk}

	fmt.Printf("%v : %v , %v : %v\n",d1.data,d1.clk,d2.data,d2.clk)
	reconciliation(d1,d2)
	fmt.Println("================================")

	fmt.Println("==TEST 6==")
	clk = make(map[string]int)
	clk["A0"] = 1
	clk["B0"] = 1
	d1 = pkt{"d1", clk}

	clk = make(map[string]int)
	clk["A0"] = 2
	clk["B0"] = 2
	d2 = pkt{"d2", clk}

	fmt.Printf("%v : %v , %v : %v\n",d1.data,d1.clk,d2.data,d2.clk)
	reconciliation(d1,d2)
	fmt.Println("================================")

	fmt.Println("==TEST 7==")
	clk = make(map[string]int)
	clk["A0"] = 1
	clk["B0"] = 4
	d1 = pkt{"d1", clk}

	clk = make(map[string]int)
	clk["A0"] = 1
	clk["B0"] = 1
	d2 = pkt{"d2", clk}

	fmt.Printf("%v : %v , %v : %v\n",d1.data,d1.clk,d2.data,d2.clk)
	reconciliation(d1,d2)
	fmt.Println("================================")

	fmt.Println("==TEST 8==")
	clk = make(map[string]int)
	clk["A0"] = 3
	clk["B0"] = 2
	d1 = pkt{"d1", clk}

	clk = make(map[string]int)
	clk["A0"] = 4
	clk["B0"] = 1
	d2 = pkt{"d2", clk}

	fmt.Printf("%v : %v , %v : %v\n",d1.data,d1.clk,d2.data,d2.clk)
	reconciliation(d1,d2)
	fmt.Println("================================")

	fmt.Println("==TEST 9==")
	clk = make(map[string]int)
	clk["A0"] = 1
	clk["B0"] = 2
	d1 = pkt{"d1", clk}

	clk = make(map[string]int)
	clk["A0"] = 2
	clk["B0"] = 1
	d2 = pkt{"d2", clk}

	fmt.Printf("%v : %v , %v : %v\n",d1.data,d1.clk,d2.data,d2.clk)
	reconciliation(d1,d2)
	fmt.Println("================================")
}

/*
Links:
https://yourbasic.org/golang/delete-element-slice/
https://stackoverflow.com/questions/9251234/go-append-if-unique
https://yourbasic.org/golang/sort-map-keys-values/
https://stackoverflow.com/questions/18343208/how-do-i-reverse-sort-a-slice-of-integer-go
https://stackoverflow.com/questions/13190836/is-there-a-way-to-iterate-over-a-slice-in-reverse-in-go
*/


