package ServerUtils

import (
	"50.041-DistSysProject-BunshinDB/pkg/ConHash"
	"50.041-DistSysProject-BunshinDB/pkg/ConHttp"
	"50.041-DistSysProject-BunshinDB/pkg/Utils"
	"50.041-DistSysProject-BunshinDB/config"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)


const FAINT_NODE_ENDPOINT = config.FAINT_NODE_ENDPOINT
const REMOVE_NODE_ENDPOINT = config.REMOVE_NODE_ENDPOINT
const REVIVE_NODE_ENDPOINT = config.REVIVE_NODE_ENDPOINT
//Once we exceed 10, we will declare it as a "permanent failure"

//used only for to store and to marshal out
type NodeStatusRestOutput struct {
	StatusArray []map[string] string
}

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

	// {nodeID: numberOfTimesItHasFailed }
	nodeStatuses map[string]int
	failThreshold int
}

func (s *StethoNode) addNode(nodeID string, nodeAddr string){

	if _, ok := s.nodeStatuses[nodeID]; ok {
		if s.nodeStatuses[nodeID] != -1 {
			fmt.Printf("[STETHO] Node %s already exists in Stetho's record. Will not duplicate.\n")
			return
		} else {
			fmt.Printf("[STETHO] Node %s - detected previous perm failure. Adding anyway...")
		}
	}

	s.nodeStatuses[nodeID] = 0
 	s.nodeInfoArray = append(s.nodeInfoArray, NodeInfo{
		id:  nodeID,
		url: nodeAddr,
	})

}

func (s *StethoNode) SetRing(ringAddr string){
	//s.ringServer = r
	s.ringAddr = ringAddr
}

func (s *StethoNode) ping(nodeID string, urlString string){
	resp, err := s.client.Get(urlString)
	if err != nil {
		log.Printf("[STETHO] Failed to pingAll %s at %s because of error: %s", nodeID, urlString, err)
		s.handleFailedNode(nodeID)
	} else {

		if resp.StatusCode == 200{
			//success case
			if s.nodeStatuses[nodeID] > 0 {
				//	fainted - now we revive
				s.nodeStatuses[nodeID] = 0 //reset
				fmt.Printf("[STETHO] Node %s has revived! \n", nodeID)
				go s.postToRingServer(nodeID, REVIVE_NODE_ENDPOINT)

			}
			//TODO: put one more if else here add call revive-node conditionally
			fmt.Printf("[STETHO] Node %s is alive \n", nodeID )
		}
	}
}


func (s *StethoNode) pingAll(){
	time.Sleep(time.Duration(1 * time.Second))
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

			go s.ping(nodeID, urlString)

		}
		time.Sleep(time.Duration(1 * time.Second))
	}
}

func (s *StethoNode) handleFailedNode(nodeID string){
	//First faint
	if s.nodeStatuses[nodeID] == 0 {
		s.faintNode(nodeID)
		s.nodeStatuses[nodeID] +=1

	} else if s.nodeStatuses[nodeID] < s.failThreshold{
		//the case for 1 - 9
		s.nodeStatuses[nodeID] +=1
	} else if s.nodeStatuses[nodeID] >= s.failThreshold {
		s.removeNode(nodeID)
	}
	fmt.Println(s.nodeStatuses)
}

func (s *StethoNode) faintNode(nodeId string){

	go s.postToRingServer(nodeId, FAINT_NODE_ENDPOINT)
}

func (s *StethoNode) removeNode(nodeId string) {
	log.Printf("[Stetho] Removing [Node %s] due to perm failure \n", nodeId)

	//Set to -1 so we can see later
	s.nodeStatuses[nodeId] = - 1
	//TODO: need to make sure i do not append duplicates
	for i, nodeInfo := range s.nodeInfoArray {
		if nodeInfo.id == nodeId {
			s.nodeInfoArray = append(s.nodeInfoArray[:i], s.nodeInfoArray[i+1:]...)
			break
		}
	}
	log.Println("After Removing: ", s.nodeInfoArray)
	go s.postToRingServer(nodeId, REMOVE_NODE_ENDPOINT)
}


func (s *StethoNode) postToRingServer(nodeId string, endpoint string){
	//REVIVE, FAINT, REMOVE
	postUrl := fmt.Sprintf("http://%s/%s", s.ringAddr, endpoint)
	log.Println(postUrl)
	requestBody, err := json.Marshal(map[string]string {
		//TODO: don't hardcode it
		"nodeId": nodeId,
	})

	if err != nil {
		log.Println(err)
	}

	//TODO: Explore refactoring the below lines
	_, err = http.Post(postUrl, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println("Check if Ring Server is up and running")
		log.Println(err)
	}

}


func (s *StethoNode) HttpServerStart(){

	http.HandleFunc("/set-ring", s.SetRingHandler)
	http.HandleFunc("/add-node", s.AddNodeHandler)
	http.HandleFunc("/get-status", s.GetStatusHandler)
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
		log.Println(err)
	}
	//log.Println(string(body))

	var payload map[string]string
	err = json.Unmarshal(body, &payload)

	if err != nil {
		log.Println(err)
	}
	s.addNode(payload["nodeID"], payload["nodeUrl"])
	log.Println("[STETHO] After receiving the post request ", s.nodeInfoArray)

}

func (s *StethoNode) Start(){
	go s.HttpServerStart()
	//Not using a go routine here so that it blocks
	s.pingAll()

}

func (s *StethoNode) GetStatusHandler(w http.ResponseWriter, r *http.Request) {
	Utils.EnableCors(&w)
	//start with empty array
	res := NodeStatusRestOutput{StatusArray: [] map[string] string{}}
	for k, v := range s.nodeStatuses {
		newNodeStatus := map[string]string {}
		newNodeStatus["name"] = k

		switch v {
		case -1:
			newNodeStatus["status"] = "-1"
		case 0:
			newNodeStatus["status"] = "0"
		default:
			newNodeStatus["status"] = "1"
		}
		res.StatusArray = append(res.StatusArray, newNodeStatus)

		}


	sort.Slice(res.StatusArray,
		func(i, j int) bool {
			return res.StatusArray[i]["name"] < res.StatusArray[j]["name"]
		})

	body, err := json.Marshal(res)
	if err != nil {
		log.Println(err)
	}

	_, err = w.Write(body)

	if err != nil {
		log.Println(err)
	}

}

func NewStethoServer(port string, numSeconds int, timeoutSeconds int, failThreshold int) StethoNode {
	client := http.Client{Timeout:time.Duration(time.Duration(timeoutSeconds) * time.Second)}

	nodeInfoArray := []NodeInfo{}
	ringServer := ""
	nodeStatuses := map[string] int {}

	return StethoNode{client, nodeInfoArray,
		ringServer, port, numSeconds, nodeStatuses, failThreshold}


}



