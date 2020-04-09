package Stetho

import (
	"50.041-DistSysProject-BunshinDB/pkg/ConHash"
	"50.041-DistSysProject-BunshinDB/pkg/ConHttp"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type StethoNode struct {

	client http.Client
	nodes []ConHash.Node
	ringServer ConHttp.Ring
	port string
	pingIntervalSeconds int //How long the stethoscope should wait after each cycle?
}

func (s *StethoNode) AddNode(n ConHash.Node){
	s.nodes = append(s.nodes, n)

}

func (s *StethoNode) SetRing(r ConHttp.Ring){
	s.ringServer = r
}

//TODO: Can explore making ping() async. Ping shall be synchronous for now.
func (s *StethoNode) ping(){
	time.Sleep(time.Duration(5 * time.Second))
	log.Print("Stetho is up and pinging")
	for {
		for _, node := range(s.nodes){
			//https://github.com/golang/go/issues/18824
			urlString := fmt.Sprintf("http://%s:%s/%s", node.IP, node.Port, "hb")
			log.Print(fmt.Sprintf("Pinging %s at %s",
				node.CName, urlString))

			resp, err := s.client.Get(urlString)

			//Fails for some reason
			//TODO: need to be able to differentiate the type of failure such as timeout vs no host vs invalid port etc.
			if err != nil {

			}

			log.Println(resp.StatusCode)
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatalln(err)
			}

			log.Println(string(body))
		}
		time.Sleep(time.Duration(1 * time.Second))
	}
}

func (s *StethoNode) HttpServerStart(){

	http.HandleFunc("/set-ring", s.SetRingHandler)
	http.HandleFunc("/add-node", s.AddNodeHandler)
	ip, err := ConHttp.ExternalIP()

	if err == nil {
		log.Print(fmt.Sprintf("StethoNode Node listening at %s.", ip ))
	} else {
		log.Print(err)
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s",s.port), nil))
}

func (s *StethoNode) SetRingHandler(w http.ResponseWriter, r *http.Request) {
	//To-Do update ring
	//Need a onUpdateRing function in conHash.go
	log.Print("set-ring")
}

func (s *StethoNode) AddNodeHandler(w http.ResponseWriter, r *http.Request) {
	//To-Do update ring
	//Need a onUpdateRing function in conHash.go
	log.Print("add-node")
}

func (s *StethoNode) Start(){
	s.ping()
	s.HttpServerStart()

}

func NewStetho(port string, numSeconds int, timeoutSeconds int) StethoNode {
	client := http.Client{Timeout:time.Duration(time.Duration(timeoutSeconds) * time.Second)}
	nodes := []ConHash.Node{}
	dummyRing := ConHttp.Ring{}
	return StethoNode{client, nodes, dummyRing, port, numSeconds}
}



