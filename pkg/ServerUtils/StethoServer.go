package ServerUtils

import (
	"50.041-DistSysProject-BunshinDB/pkg/ConHash"
	"50.041-DistSysProject-BunshinDB/pkg/ConHttp"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type NodeInfo struct {
	id string 
	url string
}

type StethoNode struct {

	client              http.Client
	nodeInfoArray       []NodeInfo
	ringAddr            string
	port                string
	pingIntervalSeconds int //How long the stethoscope should wait after each cycle?
}

func (s *StethoNode) AddNode(nodeID string, nodeAddr string){
	//s.nodes = append(s.nodes, n)
	s.nodeInfoArray = append(s.nodeInfoArray, NodeInfo{
		id:  nodeID,
		url: nodeAddr,
	})

}

func (s *StethoNode) SetRing(ringAddr string){
	//s.ringServer = r
	s.ringAddr = ringAddr
}

//TODO: Can explore making ping() async. Ping shall be synchronous for now.
func (s *StethoNode) ping(){
	time.Sleep(time.Duration(5 * time.Second))
	log.Print("Stetho is up and pinging")
	for {
		for _, nodeInfo := range(s.nodeInfoArray){
			//https://github.com/golang/go/issues/18824
			nodeID := nodeInfo.id
			nodeAddr := nodeInfo.url
			urlString := fmt.Sprintf("http://%s/%s", nodeAddr, "hb")
			//log.Print(fmt.Sprintf("Pinging %s at %s",
			//	node.CName, urlString))
			log.Print(fmt.Sprintf("[STETHO] Pinging %s at %s", nodeID, urlString))

			resp, err := s.client.Get(urlString)

			//Fails for some reason
			//TODO: need to be able to differentiate the type of failure such as timeout vs no host vs invalid port etc.
			if err != nil {
				log.Printf("[STETHO] Failed to ping %s at %s because of error: %s", nodeID, nodeAddr, err)
			} else {
				if resp.StatusCode == 200{
					fmt.Println("ALIVE: ", nodeAddr )
				}
			}

			time.Sleep(time.Duration(5 * time.Second))
		}
		time.Sleep(time.Duration(1 * time.Second))
	}
}

func (s *StethoNode) HttpServerStart(){

	http.HandleFunc("/set-ring", s.SetRingHandler)
	http.HandleFunc("/add-node", s.AddNodeHandler)
	ip, err := ConHttp.ExternalIP()

	if err == nil {
		log.Print(fmt.Sprintf("StethoNode Node listening at %s:%s.", ip, s.port))
	} else {
		log.Print(err)
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s",s.port), nil))
}

//TODO: TEMPORARY, TO BE REMOVED
type Ring struct{
	ConHash.Ring
	stethoUrl string
}


// Request.RemoteAddress contains port, which we want to remove i.e.:
// "[::1]:58292" => "[::1]"
func ipAddrFromRemoteAddr(s string) string {
	idx := strings.LastIndex(s, ":")
	if idx == -1 {
		return s
	}
	return s[:idx]
}

// requestGetRemoteAddress returns ip address of the client making the request,
// taking into account http proxies
func requestGetRemoteAddress(r *http.Request) string {
	hdr := r.Header
	hdrRealIP := hdr.Get("X-Real-Ip")
	hdrForwardedFor := hdr.Get("X-Forwarded-For")
	if hdrRealIP == "" && hdrForwardedFor == "" {
		return ipAddrFromRemoteAddr(r.RemoteAddr)
	}
	if hdrForwardedFor != "" {
		// X-Forwarded-For is potentially a list of addresses separated with ","
		parts := strings.Split(hdrForwardedFor, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		// TODO: should return first non-local address
		return parts[0]
	}
	return hdrRealIP
}


func (s *StethoNode) SetRingHandler(w http.ResponseWriter, r *http.Request) {
	//To-Do update ring
	//Need a onUpdateRing function in conHash.go
	log.Print("[STETHO] Receiving Ring Registration...")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalln(err)
	}
	var payload map[string]string
	err = json.Unmarshal(body, &payload)

	if err != nil {
		log.Fatalln(err)
	}
	s.ringAddr = fmt.Sprintf("%s:%s", requestGetRemoteAddress(r), payload["ringPort"])
	log.Println("[STETHO] After receiving the post request, Ring address is  ", s.ringAddr)

}

func (s *StethoNode) AddNodeHandler(w http.ResponseWriter, r *http.Request) {
	//To-Do update ring
	//Need a onUpdateRing function in conHash.go
	log.Print("add-node")

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	//log.Println(string(body))

	var payload map[string]string
	err = json.Unmarshal(body, &payload)

	if err != nil {
		log.Fatalln(err)
	}
	s.AddNode(payload["nodeID"], payload["nodeUrl"])
	log.Println("[STETHO] After receiving the post request ", s.nodeInfoArray)
}

func (s *StethoNode) Start(){
	go s.HttpServerStart()
	//Not using a go routine here so that it blocks
	s.ping()

}

func NewStethoServer(port string, numSeconds int, timeoutSeconds int) StethoNode {
	client := http.Client{Timeout:time.Duration(time.Duration(timeoutSeconds) * time.Second)}

	nodeInfoArray := []NodeInfo{}
	ringServer := ""

	return StethoNode{client, nodeInfoArray, ringServer, port, numSeconds}


}



