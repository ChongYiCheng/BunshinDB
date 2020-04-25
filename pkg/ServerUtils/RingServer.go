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

type RingVisualSegment struct {
	ID    string
	CName string
	//A list with 2 element [p1, p2] refers to a range of (p1, p2]
	PosRange [2]int
	//Length = PosRange[1] - PosRange[0]
	Length int
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
	http.HandleFunc("/get-ring", ringServer.GetRingHandler)
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

	ringServer.ring.NodeStatuses[nodeID] = true
	ringServer.RegisterNodeWithStetho(nodeID, nodeUrl)
	ringServer.onRingChange(true)
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

	ringServer.onRingChange(false)

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
	nodeId := payload["nodeId"]
	delete(ringServer.ring.NodeStatuses, nodeId)
	ringServer.removeNodeFromRing(string(nodeId[0]))

	fmt.Println("New Status Map ", ringServer.ring.NodeStatuses)
	ringServer.onRingChange(true)
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
	ringServer.onRingChange(false)
}


func (ringServer RingServer) GetRingHandler(w http.ResponseWriter, r *http.Request) {
	//Marshal into json and then return
	//body, err := json.Marshal(ringServer.ring)
	ret := []RingVisualSegment{}
	var prevPos int
	for i, nodeData := range ringServer.ring.RingNodeDataArray {
		if nodeData.ID == "" {continue}
		newRVS := RingVisualSegment{
			ID:       nodeData.ID,
			CName:    nodeData.CName,
			PosRange: [2]int{prevPos, i},
			Length:   i - prevPos,
		}
		ret = append(ret, newRVS)
		prevPos = i
	}

	fmt.Println(ret)

	body, err := json.Marshal(ret)
	if err != nil {
		log.Println(err)
	}
	i, err := w.Write(body)
	fmt.Println(i)
}

func (ringServer RingServer) GetNodeHandler(w http.ResponseWriter, r *http.Request) {
	//TODO: Implement this for stronger consistency gurantees - but is anybody even calling it
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

func (ringServer RingServer) RegisterWithStetho(ringServerPort string, endpoint string) {
	postUrl := fmt.Sprintf("%s/%s", ringServer.stethoUrl, endpoint)
	requestBody, err := json.Marshal(map[string]string {
		//TODO: don't hardcode it
		"ringPort": ringServerPort,
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

func (ringServer *RingServer) onRingChange(shouldGenPrefList bool) {

	if shouldGenPrefList {
		fmt.Printf("Pref List BC: %s \n", ringServer.ring.NodePrefList)
		ringServer.ring.GenPrefList()
		fmt.Printf("Pref List AC: %s \n", ringServer.ring.NodePrefList)
	}

	ringServer.updateRing()
}


func (ringServer *RingServer) updateRing(){
	for _, nodeData := range ringServer.ring.RingNodeDataArray{
		//TODO: investigate why url is empty
		if nodeData.IP == "" || nodeData.ID[len(nodeData.ID) - 1] != '0' {
			continue
		}
		nodeUrl := fmt.Sprintf("%s:%s", nodeData.IP, nodeData.Port)
		postUrl := fmt.Sprintf("http://%s/%s", nodeUrl, NEW_RING_ENDPOINT)
		fmt.Printf("Sending New Ring to Node %s at %s \n", nodeData.ID, postUrl)
		requestBody, err := json.Marshal(ringServer.ring)
		if err != nil {
			fmt.Println(err)
			return
		}
		go http.Post(postUrl, "application/json", bytes.NewBuffer(requestBody))
	}

	fmt.Println("Done updating ring")
}

func (ringServer *RingServer) removeNodeFromRing(cName string) {
	fmt.Println("[RingServer] Removing Node From Ring...")
	for i, nodeData := range ringServer.ring.RingNodeDataArray{
		if nodeData.CName == cName {
			//Considered a deletion
			fmt.Println("Deleting ", nodeData)
			ringServer.ring.RingNodeDataArray[i] = ConHash.NodeData{}
		}
	}
	fmt.Println("After removal: ", ringServer.ring.RingNodeDataArray)
}




func main(){
	fmt.Println()
}