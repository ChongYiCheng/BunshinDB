package main

import (
	"50.041-DistSysProject-BunshinDB/pkg/Utils"
	"50.041-DistSysProject-BunshinDB/pkg/ShoppingCart"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"io/ioutil"
	"log"

)

type Message struct{
    SenderIP string
    SenderPort string
    Data map[string][]byte //Key-Value pair
    Query string //Just a key string for receiver to query
}

type Client struct{
    IP string
    Port string
    KnownNodeURLs []string
}

func (client *Client) HttpServerStart(){

    http.HandleFunc("/get", client.GetHandler)
    http.HandleFunc("/put", client.PutHandler)
    log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s",client.Port), nil))
}

func (client *Client) GetHandler(w http.ResponseWriter, r *http.Request){

    fmt.Println("Client Get Handler activated")
	w.Header().Set("Access-Control-Allow-Origin", "*")

    w.Header().Set("Access-Control-Allow-Headers", "*")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	ID, ok := r.URL.Query()["ID"]

    if !ok || len(ID[0]) < 1 {
        log.Println("ID Param 'key' is missing")
        return
    }

	shopperID := ID[0]
    fmt.Printf("ShopperID is %s\n",shopperID)
    fmt.Println("Querying BunshinDB ...")

    httpMsg := &Message{}
    httpMsg.SenderIP = client.IP
    httpMsg.SenderPort = client.Port
    httpMsg.Query = shopperID
    //fmt.Printf("httpMsg %s\n",httpMsg)

	rand.Seed(time.Now().Unix())
	targetUrl := client.KnownNodeURLs[rand.Intn(len(client.KnownNodeURLs))]
    msgData, err := client.HttpClientReq(httpMsg,targetUrl,"get")
    if err != nil{
        fmt.Println(err)
        http.Error(w, "Failed to retrieve items", 500)
    }

    w.Header().Set("Content-Type","application/json")
    json.NewEncoder(w).Encode(string(msgData[shopperID]))

}

func (client *Client) PutHandler(w http.ResponseWriter, r *http.Request){

    fmt.Println("Client Put Handler activated")

	w.Header().Set("Access-Control-Allow-Origin", "*")

    w.Header().Set("Access-Control-Allow-Headers", "*")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

    if r.Body == nil {
        http.Error(w, "Please send a request body", 400)
        return
    }


    var shoppingCart ShoppingCart.ShoppingCart

    // Try to decode the request body into the struct. If there is an error,
    // respond to the client with the error message and a 400 status code
	fmt.Println(r.Body)
    err := json.NewDecoder(r.Body).Decode(&shoppingCart)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    shoppingCartJson, marshalErr := json.Marshal(shoppingCart)
    if marshalErr != nil{
        fmt.Errorf("Failed to marshal shoppingCart")
    }

    clientData := map[string][]byte{shoppingCart.ShopperID:shoppingCartJson}


    httpMsg := &Message{}
    httpMsg.SenderIP = client.IP
    httpMsg.SenderPort = client.Port
    httpMsg.Data = clientData
    //fmt.Printf("httpMsg %s\n",httpMsg)

	rand.Seed(time.Now().Unix())
	targetUrl := client.KnownNodeURLs[rand.Intn(len(client.KnownNodeURLs))]
    msgData, err := client.HttpClientReq(httpMsg,targetUrl,"put")
    if err != nil{
        http.Error(w, "Failed to put items", 500)
    }


    json.NewEncoder(w).Encode(string(msgData[shoppingCart.ShopperID]))
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")

}

func setupResponse(w *http.ResponseWriter, req *http.Request) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
    (*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
    (*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func contains(arr []string, str string) bool {
   for _, a := range arr {
      if a == str {
         return true
      }
   }
   return false
}

func (client *Client) HttpClientReq(msg *Message,targetUrl string,endpoint string) (map[string][]byte,error){
	httpClient := &http.Client{
	}
    fmt.Println("HTTP Client Req function called")
    url := fmt.Sprintf("http://%s/%s",targetUrl,endpoint)

    jsonBuffer, marshalErr := json.Marshal(msg)
    if marshalErr != nil{
        fmt.Errorf("Failed to Marshal message")
    }

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBuffer))
    req.Header.Set("Content-Type", "application/json")

    res, err := httpClient.Do(req)
    if err != nil{
        fmt.Printf("Cannot reach server at %v\n",url)
        unreachableUrls := []string{}
        unreachableUrls = append(unreachableUrls,targetUrl)
        for contains(unreachableUrls,targetUrl){
            rand.Seed(time.Now().UTC().Unix())
            dstNodeidx := rand.Intn(len(client.KnownNodeURLs))
            fmt.Printf("Client sending to Node %s\n",client.KnownNodeURLs[dstNodeidx])
            targetUrl = client.KnownNodeURLs[dstNodeidx]
        }
        return client.HttpClientReq(msg, targetUrl, endpoint)
        //return map[string][]byte{}, err
    }
    defer res.Body.Close()
    fmt.Println("HTTP Client Req - Got a response")


    // always close the response-body, even if content is not required

    var resMsg Message
    json.NewDecoder(res.Body).Decode(&resMsg)
    msgData := map[string]ShoppingCart.ShoppingCart{}
    if endpoint == "get" && len(msgData) > 1{
        //There are conflicting shopping cart versions, client to perform semantic reconciliation and write back to coordinator
        reconciledData := client.SemanticReconciliation(resMsg)
        return reconciledData, nil
    } else{
        msgData := resMsg.Data
        for _,v := range resMsg.Data{
            var shoppingCart ShoppingCart.ShoppingCart
            unMarshalErr := json.Unmarshal(v,&shoppingCart)
            if unMarshalErr != nil{
                fmt.Errorf("Failed to unmarshal message data")
            }
        }

        return msgData, nil
    }
}

func (client *Client) SemanticReconciliation(conflictedMessage Message) (map[string][]byte){
    //Need to collate list of conflicted shopping carts then merge them
    fmt.Println("Client running Semantic Reconciliation")
    listOfConflictingCarts := []ShoppingCart.ShoppingCart{}
    msgData := conflictedMessage.Data
    for _,v := range msgData{
        var shoppingCart ShoppingCart.ShoppingCart
        unMarshalErr := json.Unmarshal(v,&shoppingCart)
        if unMarshalErr != nil{
            fmt.Errorf("Failed to unmarshal message data")
        }
        listOfConflictingCarts = append(listOfConflictingCarts,shoppingCart)
    }
    reconciledCart := ShoppingCart.MergeShoppingCarts(listOfConflictingCarts)
    key := reconciledCart.ShopperID
    rawValue := reconciledCart
    value, marshalErr := json.Marshal(rawValue)
    if marshalErr != nil{
        fmt.Errorf("Failed to marshal message")
    }
    data := map[string][]byte{key:value}
    httpMsg := &Message{}
    httpMsg.SenderIP = client.IP
    httpMsg.SenderPort = client.Port
    httpMsg.Data = data
    fmt.Printf("httpMsg %s\n",httpMsg)
    rand.Seed(time.Now().Unix())
    targetUrl := client.KnownNodeURLs[rand.Intn(len(client.KnownNodeURLs))]
    msgData,err := client.HttpClientReq(httpMsg,targetUrl,"put")
    if err!=nil{
        fmt.Errorf("Failed to put data")
    }
    return msgData
}

func (client *Client) runCommand(commandStr string) error {
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr, parseErr := Utils.ParseCommandLine(commandStr)
    if parseErr != nil{
        fmt.Errorf("Failed to parse user command inputs")
    }

    //Subcommands
    if len(arrCommandStr)>=1{
		switch arrCommandStr[0] {
		case "exit":
			os.Exit(0)
			// add another case here for custom commands.
        case "help":
            fmt.Printf(
`
Here are the list of commands:

help: Shows list of commands

exit: quits program

get: Usage - get <key>
query will send a key to a random DynamoDB Node and retrieves the key-value pair
from the Coordinator Node 

put: Usage - put <json file>
put will send a shopping cart structure to a random DyanmoDB Node and put it in the database
under the Coordinator Node
`)
		case "get":
            if len(arrCommandStr)!=2{
                return fmt.Errorf("Usage of query - query <Key>")
            }
            httpMsg := &Message{}
            httpMsg.SenderIP = client.IP
            httpMsg.SenderPort = client.Port
            key := arrCommandStr[1]
            httpMsg.Query = key
            fmt.Printf("httpMsg %s\n",httpMsg)
            rand.Seed(time.Now().Unix())
            dstNodeidx := rand.Intn(len(client.KnownNodeURLs))
            fmt.Printf("Client sending to Node %s\n",client.KnownNodeURLs[dstNodeidx])
			targetUrl := client.KnownNodeURLs[dstNodeidx]
            client.HttpClientReq(httpMsg,targetUrl,"get")

        case "put":
			if len(arrCommandStr)!=2{
			   return fmt.Errorf("Usage of put - put <json file>")
			}
			httpMsg := &Message{}
			httpMsg.SenderIP = client.IP
			httpMsg.SenderPort = client.Port
            content, err := ioutil.ReadFile(arrCommandStr[1])
            if err != nil{
                fmt.Errorf("Error trying to read JSON file")
            }

            var shoppingCart ShoppingCart.ShoppingCart
            unmarshalErr := json.Unmarshal(content, &shoppingCart)
            if unmarshalErr != nil{
                fmt.Errorf("Failed to unmarshal content from json file into a shopping cart")
            }

			key := shoppingCart.ShopperID
			rawValue := shoppingCart
			value, marshalErr := json.Marshal(rawValue)
            if marshalErr != nil{
                fmt.Errorf("Failed to marshal message")
            }
			data := map[string][]byte{key:value}
			httpMsg.Data = data
			fmt.Printf("httpMsg %s\n",httpMsg)
            rand.Seed(time.Now().Unix())
            dstNodeidx:= rand.Intn(len(client.KnownNodeURLs))
            fmt.Printf("Client sending to Node %s\n",client.KnownNodeURLs[dstNodeidx])
            targetUrl := client.KnownNodeURLs[dstNodeidx]
			client.HttpClientReq(httpMsg,targetUrl,"put")
        default:
		cmd := exec.Command(arrCommandStr[0], arrCommandStr[1:]...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		return cmd.Run()
    }
}
    return nil
}

func main(){
	if len(os.Args) != 2{
        fmt.Printf("Usage of program is: %s <PORT>\n",os.Args[0])
        os.Exit(0)
    }
    currentIP, err := Utils.ExternalIP()
    fmt.Printf("Setting Node's IP to be %s\n",currentIP)
    if err != nil{
        fmt.Errorf("Failed to obtain IP address")
    }
    port := os.Args[1]

    KnownNodeUrls := []string{fmt.Sprintf("%s:8080",currentIP),fmt.Sprintf("%s:8081",currentIP),fmt.Sprintf("%s:8082",currentIP),fmt.Sprintf("%s:8083",currentIP)}
    client := &Client{currentIP,port,KnownNodeUrls}
    go client.HttpServerStart()
	//Start of CLI interactivity
	reader := bufio.NewReader(os.Stdin)
    fmt.Printf("Client@%s:%s$ ",client.IP,client.Port)
	for {
        fmt.Printf("Client@%s:%s$ ",client.IP,client.Port)
		cmdString, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		err = client.runCommand(cmdString)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
