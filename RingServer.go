package main

import (
	"50.041-DistSysProject-BunshinDB/pkg/ConHash"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

//TODO: Decide on standard port number for ring in future, currently hardcoded
func (ring Ring) HttpServerStart(){
	http.HandleFunc("/add-node", ring.AddNodeHandler)
	http.HandleFunc("/faint-node", ring.FaintNodeHandler)
	http.HandleFunc("/get-node", ring.GetNodeHandler)
	http.HandleFunc("/hb", ring.HeartBeatHandler)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", "5001"), nil))
}

func (ring Ring) AddNodeHandler(w http.ResponseWriter, r *http.Request) {
	//TODO: change this
	ring.HeartBeatHandler(w, r)
}

func (ring Ring) FaintNodeHandler(w http.ResponseWriter, r *http.Request) {
	//TODO: change this
	ring.HeartBeatHandler(w, r)
}

func (ring Ring) GetNodeHandler(w http.ResponseWriter, r *http.Request) {
	//TODO: change this
	ring.HeartBeatHandler(w, r)
}

//TODO: Refactor this part
func (ring Ring) HeartBeatHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK) //Set response code to 200
	fmt.Fprintf(w,"") //Just send a blank reply at least the server knows you're reachable
}

func (ring *Ring) Start(){

	ring.HttpServerStart()
}

func (ring Ring) RegisterWithStetho( endpoint string) {
	requestBody, err := json.Marshal(map[string]string {
		//TODO: don't hardcode it
		"ringPort": "5001",
	})

	if err != nil {
		log.Fatalln(err)
	}

	postUrl := fmt.Sprintf("%s/%s", ring.stethoUrl, endpoint)
	resp, err := http.Post(postUrl, "application/json", bytes.NewBuffer(requestBody))

	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(string(body))
}

func (ring *Ring) RegisterNodeWithStetho(nodeUrl string, registerNodeEndpoint string) {
	log.Println("Registering Node with Stetho Node")
	postUrl := fmt.Sprintf("%s/%s", ring.stethoUrl, registerNodeEndpoint)
	requestBody, err := json.Marshal(map[string]string {
		//TODO: don't hardcode it
		"nodeUrl": nodeUrl,
	})

	if err != nil {
		log.Fatalln(err)
	}

	resp, err := http.Post(postUrl, "application/json", bytes.NewBuffer(requestBody))

	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(string(body))
}

func NewRingServer(conRing ConHash.Ring, stethoUrl string) Ring{
	return Ring{conRing, stethoUrl}
}