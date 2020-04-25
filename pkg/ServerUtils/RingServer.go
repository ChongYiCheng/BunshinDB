package ServerUtils

import (
	"50.041-DistSysProject-BunshinDB/pkg/ConHash"
	"50.041-DistSysProject-BunshinDB/pkg/ConHttp"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

type RingServer struct {
	ring ConHash.Ring
	stethoUrl string
	ip string
	port string
}

const ADD_NODE_URL = "add-node"
const RING_SERVER_PORT = "5001"
const NEW_RING_ENDPOINT = "new-ring"

//TODO: Decide on standard port number for ringServer in future, currently hardcoded as 5001
func (ringServer *RingServer) HttpServerStart(){
	http.HandleFunc("/add-node", ringServer.AddNodeHandler)
	http.HandleFunc("/faint-node", ringServer.FaintNodeHandler)
	http.HandleFunc("/remove-node", ringServer.RemoveNodeHandler)
	http.HandleFunc("/revive-node", ringServer.ReviveNodeHandler)
	http.HandleFunc("/get-node", ringServer.GetNodeHandler)
	http.HandleFunc("/hb", ringServer.HeartBeatHandler)
	log.Print(fmt.Sprintf("[RingServer] Started and Listening at %s:%s.", ringServer.ip, ringServer.port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", "5001"), nil))
}

func (ringServer *RingServer) AddNodeHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[RingServer] Receiving Registration from a Node")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalln(err)
	}

	var nodeDataArray []ConHash.NodeData
	err = json.Unmarshal(body, &nodeDataArray)
	if err != nil {
		log.Println(err)
	}
	phyNode := nodeDataArray[0]
	nodeID := phyNode.ID
	nodeUrl := fmt.Sprintf("%s:%s", phyNode.IP, phyNode.Port)

	actualNodeDataArray := ringServer.ring.RegisterNodes(nodeDataArray)
	fmt.Printf("Actual Node Data Array Registered %s", actualNodeDataArray)

	ringServer.RegisterNodeWithStetho(nodeID, nodeUrl)
	ringServer.updateRing()
}


func (ringServer *RingServer) FaintNodeHandler(w http.ResponseWriter, r *http.Request) {
	log.Print("[RingServer] Received Faint Node From StethoServer...")

	//TODO: refactor the below
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalln(err)
	}
	// payload = {"nodeId": "A1"}
	var payload map[string]string
	err = json.Unmarshal(body, &payload)

	if err != nil {
		log.Println(err)
	}
	
	ringServer.ring.NodeStatuses[payload["nodeId"]] = false
	fmt.Println("New Status Map ", ringServer.ring.NodeStatuses)

	ringServer.updateRing()

}


func (ringServer *RingServer) RemoveNodeHandler(w http.ResponseWriter, r *http.Request) {
	//TODO: Merkle tree stuff
	log.Print("[RingServer] Received Remove Node From StethoServer...")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalln(err)
	}
	// payload = {"nodeId": "A1"}
	var payload map[string]string
	err = json.Unmarshal(body, &payload)

	if err != nil {
		log.Println(err)
	}

	delete(ringServer.ring.NodeStatuses, payload["nodeId"])
	//TODO: also remove from the actual ring...
	fmt.Println("New Status Map ", ringServer.ring.NodeStatuses)
	ringServer.updateRing()
}

func (ringServer *RingServer) ReviveNodeHandler(w http.ResponseWriter, r *http.Request) {
	//TODO: Hinted Handoff stuff
	log.Print("[RingServer] Received Revive Node From StethoServer...")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatalln(err)
	}
	// payload = {"nodeId": "A1"}
	var payload map[string]string
	err = json.Unmarshal(body, &payload)

	if err != nil {
		log.Println(err)
	}

	ringServer.ring.NodeStatuses[payload["nodeId"]] = true
	ringServer.updateRing()
}




func (ringServer RingServer) GetNodeHandler(w http.ResponseWriter, r *http.Request) {
	//TODO: change this
	ringServer.HeartBeatHandler(w, r)
}

//TODO: Refactor this part
func (ringServer RingServer) HeartBeatHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK) //Set response code to 200
	fmt.Fprintf(w,"") //Just send a blank reply at least the server knows you're reachable
}

func (ringServer *RingServer) Start(){
	ip, err := ConHttp.ExternalIP()
	if err != nil {
		log.Fatalln(err)
	}
	ringServer.ip = ip
	ringServer.port = RING_SERVER_PORT
	ringServer.HttpServerStart()
}

func (ringServer RingServer) RegisterWithStetho( endpoint string) {
	postUrl := fmt.Sprintf("%s/%s", ringServer.stethoUrl, endpoint)
	requestBody, err := json.Marshal(map[string]string {
		//TODO: don't hardcode it
		"ringPort": "5001",
	})

	if err != nil {
		log.Fatalln(err)
	}

	body, err := ringServer.postToStetho(postUrl, bytes.NewBuffer(requestBody))

	if err != nil {
		log.Fatalln(err)
	}

	log.Println(string(body))
}

func (ringServer *RingServer) postToStetho(reqUrl string, request io.Reader) ([]byte, error){

	resp, err := http.Post(reqUrl, "application/json", request)
	if err != nil {
		log.Println("Check if Stetho Server is up and running")
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	return body, err

}

func (ringServer *RingServer) RegisterNodeWithStetho(nodeID string, nodeUrl string) {
	log.Println("Registering Node with Stetho Node")
	postUrl := fmt.Sprintf("%s/%s", ringServer.stethoUrl, ADD_NODE_URL)
	requestBody, err := json.Marshal(map[string]string {
		//TODO: don't hardcode it
		"nodeUrl": nodeUrl,
		"nodeID": nodeID,
	})

	if err != nil {
		log.Fatalln(err)
	}

	body, err := ringServer.postToStetho(postUrl,  bytes.NewBuffer(requestBody))
	fmt.Printf("[RingServer] After Registering: %x \n", body)

}

func NewRingServer(conRing ConHash.Ring, stethoUrl string, port string) RingServer{
	ip, err := ConHttp.ExternalIP()
	if err == nil {
		return RingServer{conRing, stethoUrl, ip, port}
	} else {
		fmt.Println(err)
		log.Fatalln(err)
		return RingServer{}
	}
}

 func (ringServer *RingServer) updateRing(){
 	fmt.Printf("call update ring")
	for _, nodeData := range ringServer.ring.RingNodeDataArray{
		//TODO: investigate why url is empty
		if nodeData.IP == "" || nodeData.ID[len(nodeData.ID) - 1] != '0' {
			continue
		}
		nodeUrl := fmt.Sprintf("http://%s:%s", nodeData.IP, nodeData.Port)
		postUrl := fmt.Sprintf("http://%s/%s", nodeUrl, NEW_RING_ENDPOINT)
		fmt.Printf("Sending New Ring to Node %s at %s \n", nodeData.ID, postUrl)
		requestBody, err := json.Marshal(ringServer.ring)
		if err != nil {
			fmt.Println(err)
			return
		}
		go http.Post(postUrl, "application/json", bytes.NewBuffer(requestBody))

	}
}

func main(){
	fmt.Println()
}