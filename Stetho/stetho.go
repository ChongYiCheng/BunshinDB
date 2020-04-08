package Stetho

import (
	"50.041-DistSysProject-BunshinDB/ConHash"
	"fmt"
	"log"
	"net/http"
)

type Stetho struct {

	nodes []ConHash.Node
	ringServer Ring
	port int
}

func (s *Stetho) addNode(n *ConHash.Node){
	s.nodes = append(s.nodes, *n)

}

func (s *Stetho) addRing(r *Ring){
	s.ringServer = *r
}


func (s *Stetho) listen(){
	log.Print("listening")
	//for {
	//	for _, node := range(s.nodes){
	//		fmt.Println(node.IP)
	//	}
	//	fmt.Println("lala")
	//}
}

func (s *Stetho) HttpServerStart(){

	http.HandleFunc("/set-ring", s.SetRingHandler)
	http.HandleFunc("/add-node", s.AddNodeHandler)
	log.Print(fmt.Sprintf("Stetho Node listening at %s.",externalIP()))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s",s.port), nil))
}

func (s *Stetho) SetRingHandler(w http.ResponseWriter, r *http.Request) {
	//To-Do update ring
	//Need a onUpdateRing function in conHash.go
	log.Print("set-ring")
}

func (s *Stetho) AddNodeHandler(w http.ResponseWriter, r *http.Request) {
	//To-Do update ring
	//Need a onUpdateRing function in conHash.go
	log.Print("add-node")
}

func (s *Stetho) start(){
	s.HttpServerStart()
	s.listen()
}

func NewStetho(port int) Stetho{
	nodes := []ConHash.Node {}
	return Stetho{nodes, nil, port}
}


func main() {

	var s Stetho = NewStetho(5000)
	s.start()

}


