package main

import (
    "50.041-DistSysProject-BunshinDB/pkg/Utils"
    "50.041-DistSysProject-BunshinDB/pkg/ConHash"
    "50.041-DistSysProject-BunshinDB/pkg/ShoppingCart"
    "50.041-DistSysProject-BunshinDB/config"
    "bufio"
    "bytes"
    "encoding/json"
    "fmt"
    badger "github.com/dgraph-io/badger"
    glog "github.com/golang/glog"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "os/exec"
    "strconv"
    "strings"
    "time"
    "reflect"
)


type Message struct{
    SenderIP string
    SenderPort string
    Data map[string][]byte //Key-Value pair
    Query string //Just a key string for receiver to query

}

type Node struct{
    ConHash.Node

}

type Ring struct{
    ConHash.Ring
}


func handle(err interface{}){
	if err != nil{
		log.Fatal(err)
	}
}

func (node *Node) Start(){
    //Open the Badger database located in the node's DB Path
    // It will be created if it doesn't exist
    db ,err := badger.Open(badger.DefaultOptions(node.DBPath))
    handle(err)
    defer db.Close()
    node.NodeDB = db
    //Database for hinted handoff
    hhQueue , hhErr := badger.Open(badger.DefaultOptions(node.DBPath+"/hhQueue"))
    node.HHQueue = hhQueue
    handle(hhErr)
    defer hhQueue.Close()
    node.HttpServerStart()
}

func contains(s []int, e int) bool {
    for _, a := range s {
        if a == e {
            return true
        }
    }
    return false
}

func InPrefList(prefList []ConHash.NodeData, nodeIP string, nodePort string) bool{
    for _,nodeData := range prefList{
        if (nodeIP == nodeData.IP && nodePort == nodeData.Port){
            return true
        }
    }
    return false
}


const RING_MAX_ID = config.RING_MAX_ID
const REGISTER_ENDPOINT = config.NODESERVER_REGISTER_ENDPOINT
const WARMUP_DURATION = config.NODESERVER_WARMUP_DURATION //wait before node registers

func (n *Node) RegisterWithRingServer(ringUrl string) {
    nodeDataArray := []ConHash.NodeData {}

    for i := 0; i < n.NumTokens +1; i ++ {
        id := fmt.Sprintf("%s%d", n.CName, i)
        hash := ConHash.HashMD5(id, 0, RING_MAX_ID)
        nodeDataArray = append(nodeDataArray, ConHash.NewNodeData(id, n.CName, hash, n.IP, n.Port))
    }
    log.Println("Length: ", len(nodeDataArray))
    n.NodeDataArray = nodeDataArray
    requestBody, err := json.Marshal(nodeDataArray)
    // Send the Ring Server

    postURL := fmt.Sprintf("%s/%s", ringUrl, REGISTER_ENDPOINT)
    resp, err := http.Post(postURL, "application/json", bytes.NewReader(requestBody))
    if err != nil {
        log.Println("Check if RingServer is up and running")
        log.Fatalln(err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    //Checks response from registering with ring server. Error message comes out if fail
    fmt.Println("Response from registering w Ring Server: ", string(body))
    if err != nil {
        fmt.Println("Error at Register with Ring Server")
        log.Fatalln(err)
    }
}


func (node *Node) HttpServerStart(){

	http.HandleFunc("/get", node.GetHandler)
	http.HandleFunc("/put", node.PutHandler)
	http.HandleFunc("/new-ring", node.NewRingHandler)
	http.HandleFunc("/get-node", node.GetNodeHandler)
	http.HandleFunc("/hb", node.HeartbeatHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s",node.Port), nil))
}

func (node *Node) GetHandler(w http.ResponseWriter, r *http.Request) {
    var msg *Message
    fmt.Println("Get Handler activated")

    w.Header().Set("Content-Type", "application/json")
    if r.Body == nil {
        http.Error(w, "Please send a request body", 400)
        return
    }

    err := json.NewDecoder(r.Body).Decode(&msg)
    if err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    fmt.Println("Get Handler - Allocating Key")
    query := msg.Query
    ring := node.Ring
    dstNodeHash, _ , AllocErr := ring.AllocateKey(query)
    if AllocErr != nil{
        fmt.Printf("Failed to allocate node to key [%s]\n",query)
    }

    //Allow Nodes to check if they're in the Coordinator Node's pref list
    //If so, let them retrieve the item from their database
    if (contains(node.NodeRingPositions,dstNodeHash) || InPrefList(ring.NodePrefList[dstNodeHash],node.IP,node.Port)){ //If this node is responsible 
        fmt.Printf("[Node %s] Is Coordinator or inPrefList\n",node.CName)

        //First, try to retrieve the data from the database
        fmt.Println("Get Handler - Retrieving Key Value pair and sending it back to Requestor")
        var responseStatus string
        queryResponse, err := node.QueryDB(query)
        if err != nil{
            responseStatus = "404"
        } else {
            responseStatus = "200"
        }
        //R mechanism here - Checking to ensure R responses received before replying to client
        if ((ring.RingNodeDataArray[dstNodeHash].IP == msg.SenderIP && ring.RingNodeDataArray[dstNodeHash].Port == msg.SenderPort) ||
        InPrefList(ring.NodePrefList[dstNodeHash],msg.SenderIP,msg.SenderPort)){
            //If this request is due to the R broadcast from a coordinating node
            fmt.Printf("[Node %s] is responding to a R broadcast\n",node.CName)
            responseMessage := &Message{
                SenderIP:node.IP,SenderPort:node.Port,Data:queryResponse,
            }
            if responseStatus == "404"{
                http.Error(w,http.StatusText(http.StatusNotFound),http.StatusNotFound)
            }
            json.NewEncoder(w).Encode(responseMessage)
        } else{
            //This node has to take initiative to start the R process.
            fmt.Printf("[Node %s] is initiating an R broadcast\n",node.CName)
            otherReplicas := []ConHash.NodeData{}

            otherReplicas = append(otherReplicas,ring.RingNodeDataArray[dstNodeHash])
            otherReplicas = append(otherReplicas,ring.NodePrefList[dstNodeHash]...)

            shoppingCartVersions := []ShoppingCart.ShoppingCart{}
            if err == nil{
                var nodeShoppingCartVersion ShoppingCart.ShoppingCart
                nodeShoppingCartBytes := queryResponse[msg.Query]
                json.Unmarshal(nodeShoppingCartBytes,&nodeShoppingCartVersion)
                shoppingCartVersions = append(shoppingCartVersions,nodeShoppingCartVersion)
            }
            rChannel := make(chan Message)

            //Ask Replicas to send their version of the shoppingcart 
            rMessage := &Message{
                SenderIP:node.IP,SenderPort:node.Port,Query:msg.Query,
            }
            for _,replicaNodeData := range otherReplicas{
                if replicaNodeData.CName != node.CName{
                    physicalNodeID := replicaNodeData.CName + "0"
                    //Checks status of the target node's physical node. Skip if it has fainted(temporarily down)
                    //fmt.Printf("Status of physical Node: %t\n",node.Ring.NodeStatuses[physicalNodeID])
                    statusOfPhysicalNode := node.Ring.NodeStatuses[physicalNodeID]
                    if statusOfPhysicalNode == false{
                        fmt.Printf("Skipping node %v because it has fainted\n",replicaNodeData)
                    }else{
                        go func(rData ConHash.NodeData){
                            replicaNodeDataUrl := fmt.Sprintf("%s:%s",rData.IP,rData.Port)
                            node.HttpClientReq(rMessage,replicaNodeDataUrl,"get",rChannel)
                            responseMessage := <-rChannel
                            if len(shoppingCartVersions) < ring.RWFactor{
                                //Convert bytes in data to shopping cart structure
                                data := responseMessage.Data
                                shoppingCartBytes := data[msg.Query]
                                var shoppingCart ShoppingCart.ShoppingCart
                                json.Unmarshal(shoppingCartBytes,&shoppingCart)
                                shoppingCartVersions = append(shoppingCartVersions,shoppingCart)
                            } else{
                                // Do nothing
                            }
                        }(replicaNodeData)
                    }

                }
            }
            //Reconcile differences between the shopping carts received by doing syntactic reconciliation
            listOfConflictingShoppingCarts := ShoppingCart.CompareShoppingCarts(shoppingCartVersions)
            //If reconciliation is successful and only a single cart is left
            if len(listOfConflictingShoppingCarts) == 1{
                //Return the best version to client
                reconciledCartJson, marshalErr := json.Marshal(listOfConflictingShoppingCarts[0])
                if marshalErr != nil{
                    fmt.Errorf("Failed to marshal shopping cart")
                }
                responseData := map[string][]byte{msg.Query:[]byte(reconciledCartJson)}
                responseMessage := &Message{
                    SenderIP:node.IP,SenderPort:node.Port,Data:responseData,
                }
                json.NewEncoder(w).Encode(responseMessage)
            } else if len(listOfConflictingShoppingCarts) > 1{
                //There are multiple conflicting versions of the shopping cart
                //Need to send client the multiple conflicting versions to allow client to do semantic reconciliation
                responseData := map[string][]byte{}
                for i,shoppingCart := range listOfConflictingShoppingCarts{
                    key := strconv.Itoa(i)
                    conflictingCartJson, marshalErr := json.Marshal(shoppingCart)
                    if marshalErr != nil{
                        fmt.Errorf("Failed to marshal shopping cart")
                    }
                    responseData[key] = []byte(conflictingCartJson)
                }
                responseMessage := &Message{
                    SenderIP:node.IP,SenderPort:node.Port,Data:responseData,
                }
                json.NewEncoder(w).Encode(responseMessage)
            } else{
                //None of the replicas have this data so return 404
                http.Error(w,http.StatusText(http.StatusNotFound),http.StatusNotFound)
            }
        }
    } else{
        fmt.Printf("[Node %s] Get Handler - Relaying Key to the Coordinator Node %s\n",node.CName,ring.RingNodeDataArray[dstNodeHash].CName)
        //Fallback mechanism if Coordinator Node is not alive
        rChannel := make(chan Message)
        go func(dstNodeHash int,msgChnl chan Message, msgToSend *Message) {
            //Check status of the dst node's physical node. If down, look for next best option 
            node.CheckStatusAndSend(dstNodeHash,msgChnl,msgToSend,"get")
        }(dstNodeHash,rChannel,msg)
        responseMessage := <-rChannel
        fmt.Printf("[Node %s] Received response message from coordinator node\n",node.CName)
        json.NewEncoder(w).Encode(&responseMessage)
    }
}


func (node *Node) PutHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Printf("[Node %s] Put Handler activated\n",node.CName)
    var msg *Message

    w.Header().Set("Content-Type", "application/json")
    if r.Body == nil {
        http.Error(w, "Please send a request body", 400)
        return
    }

    err := json.NewDecoder(r.Body).Decode(&msg)
    if err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    msgData := msg.Data
    ring := node.Ring
    //fmt.Println("Put Handler - Allocating Key")
    for key, _ := range msgData{
        //ring.AllocateKey returns a destination node hash and a destination node url(not needed as hash will be used to find the url later on)
        dstNodeHash, _ , AllocErr := ring.AllocateKey(key)
        if AllocErr != nil{
            fmt.Printf("Failed to allocate node to key [%s]\n",key)
            http.Error(w, err.Error(), 400)
            return
        }
        //Allow Nodes to check if they're in the Coordinator Node's pref list
        //If so, let them retrieve the item from their database
        if (contains(node.NodeRingPositions,dstNodeHash) || InPrefList(ring.NodePrefList[dstNodeHash],node.IP,node.Port)){ //If this node is responsible 
            fmt.Printf("[Node %s] Node is Coordinator/InPrefList\n",node.CName)

            //First, try to retrieve the data from the database. First, try to retrieve the data from the database
            //Write mechanism here - To send out to the replicas and ensure W successful writes from W nodes
            if ((ring.RingNodeDataArray[dstNodeHash].IP == msg.SenderIP && ring.RingNodeDataArray[dstNodeHash].Port == msg.SenderPort) ||
            InPrefList(ring.NodePrefList[dstNodeHash],msg.SenderIP,msg.SenderPort)){
                fmt.Printf("Node %s is responding to a W broadcast by %s:%s\n",node.CName,msg.SenderIP,msg.SenderPort)
                //If this request is due to the W process started by a coordinator
                updateErr := node.UpdateDB(msgData)
                var responseStatus string
                if updateErr != nil{
                    responseStatus = "400"
                } else {
                    responseStatus = "200"
                }
                responseMessage := &Message{
                    SenderIP:node.IP,SenderPort:node.Port,Data:msgData,
                }
                if responseStatus == "400"{
                    http.Error(w, http.StatusText(http.StatusBadRequest),http.StatusBadRequest)
                } else{
                    fmt.Printf("[Node %s] Acknowledging successful put request by Coordinating Node\n",node.CName)
                    json.NewEncoder(w).Encode(responseMessage)
                }
            } else{
                fmt.Printf("[Node %s] is initiating W process\n",node.CName)
                //fmt.Printf("This is because the msg was sent by %s:%s\n",msg.SenderIP,msg.SenderPort)
                //This node has to take initiative to start the W process.
                //We need to convert the data back to a shopping cart structure
                var clientShoppingCart ShoppingCart.ShoppingCart
                json.Unmarshal(msgData[key],&clientShoppingCart)
                //Afterwards, update the version in the client shopping cart
                clientShoppingCart.Version = ShoppingCart.UpdateVersion(clientShoppingCart.Version,node.CName)
                //After that we can finally update our own database and write to the other nodes
                //Convert shopping cart back to bytes
                clientCartBytes,marshalErr := json.Marshal(clientShoppingCart)
                if marshalErr != nil{
                    fmt.Errorf("Failed to Marshal client cart")
                    http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
                    return
                }
                cartData := map[string][]byte{key:[]byte(clientCartBytes)}
                updateErr := node.UpdateDB(cartData)
                if updateErr != nil{
                    http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
                    return
                }
                writeMsg := &Message{
                    SenderIP:node.IP,SenderPort:node.Port,Data:cartData,
                }
                otherReplicas := []ConHash.NodeData{}
                otherReplicas = append(otherReplicas,ring.RingNodeDataArray[dstNodeHash])
                otherReplicas = append(otherReplicas,ring.NodePrefList[dstNodeHash]...)
                var successfulReplications = 0
                var repPointer = &successfulReplications
                wChannel := make(chan Message)
                //This sends to the other replicas
                //fmt.Printf("Other Replicas : %v\n",otherReplicas)
                for _,replicaNodeData := range otherReplicas{
                    //fmt.Printf("ReplicaNodeData is %v\n",replicaNodeData)
                    if replicaNodeData.CName != node.CName{
                        physicalNodeID := replicaNodeData.CName + "0"
                        statusOfPhysicalNode := node.Ring.NodeStatuses[physicalNodeID]
                        //Check if the destination node's physical node has fainted. false = fainted, true = alive
                        if statusOfPhysicalNode == false{
                            //If the destination node's physical node is down, save into hinted handoff database
                            fmt.Printf("Physical Node %s has failed. Saving cart into hinted handoff database\n",physicalNodeID)
                            replicaNodeHash := replicaNodeData.Hash
                            //Save replicaNodeHash,key value as hinted handoff in this node
                            node.RunHintedHandOff(replicaNodeHash,key,[]byte(clientCartBytes))
                        }else{
                            //If destination node's physical node is alive, send to it
                            fmt.Println("Proceed to send to replica")
                            go func(rData ConHash.NodeData, rcvChannel chan Message) {
                                replicaNodeDataUrl := fmt.Sprintf("%s:%s",rData.IP,rData.Port)
                                fmt.Printf("Sending replica to %s\n",replicaNodeDataUrl)
                                node.HttpClientReq(writeMsg,replicaNodeDataUrl,"put",rcvChannel)
                            }(replicaNodeData,wChannel)
                        }
                    }else{
                        //Do not request from itself
                    }
                }
                for{
                    select{
                    case <-wChannel:
                        *repPointer = *repPointer + 1
                        if successfulReplications >= ring.RWFactor{
                            fmt.Printf("[Node %s] Put operation succeeded, replying client\n",node.CName)
                            responseMessage := &Message{
                                SenderIP:node.IP,SenderPort:node.Port,Data:cartData,
                            }
                            //fmt.Printf("response message after success replication: %v\n",*responseMessage)
                            w.WriteHeader(http.StatusOK)
                            json.NewEncoder(w).Encode(responseMessage)
                            return
                        }
                    case <-time.After(500 * time.Millisecond):
                            //Return 501 code because Server failed to complete write (which means alot of failures in DB)
                            http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
                            return
                    }
                }
            }
        } else{
            fmt.Printf("[Node %s] Relaying client request to Coordinator\n",node.CName)
            rChannel := make(chan Message)
            go func(dstNodeHash int,msgChnl chan Message,msgToSend *Message) {
                node.CheckStatusAndSend(dstNodeHash,msgChnl,msgToSend,"put")
            }(dstNodeHash,rChannel,msg)
            relayResponseMsg := <-rChannel
            json.NewEncoder(w).Encode(&relayResponseMsg)
        }
    }
}

func (node *Node) NewRingHandler(w http.ResponseWriter, r *http.Request) {

    log.Printf("[Node %s] Received new ring\n", node.ID)
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        log.Fatalln(err)
    }
    //Track number of unique nodes in the previous ring
    uniqueNodes := map[string]struct{}{}
    for k,_ := range node.Ring.NodeStatuses{
        uniqueNodes[k] = struct{}{}
    }


    var ring ConHash.Ring
    err = json.Unmarshal(body, &ring)

    if err != nil {
        log.Println(err)
    }

    node.Ring = &ring
    node.NodeRingPositions = []int{}
    for _,nodeData := range node.Ring.RingNodeDataArray{
        if nodeData.CName == node.CName{
            node.NodeRingPositions = append(node.NodeRingPositions,nodeData.Hash)
        }
    }
    file, marshalIndentErr := json.MarshalIndent(ring, "", " ")
    if marshalIndentErr != nil{
        fmt.Errorf("Failed to marshal ring into JSON file")
    }
    ringJsonPath := fmt.Sprintf("/tmp/%s_RING",node.CName)
    writeFileErr := ioutil.WriteFile(ringJsonPath, file, 0644)
    if writeFileErr != nil{
        fmt.Errorf("Failed to write JSON file to path %s",ringJsonPath)
    }

    //Check Hinted Handoff against new ring and send hinted hand off if node for hinted handoff is alive
    node.CheckHintedHandOff()
    //Compare unique nodes in new ring to the previous ring.
    //If unique nodes are different, reallocate keys
    newUniqueNodes := map[string]struct{}{}
    for k,_ := range node.Ring.NodeStatuses{
        newUniqueNodes[k] = struct{}{}
    }
    if reflect.DeepEqual(uniqueNodes,newUniqueNodes) == false{
        node.ScanDB()
    }
    fmt.Printf("Updated Node Positions: %v\n",node.NodeRingPositions)

}

//Scan all key, value pairs and reallocate them
//If the KV pair belongs to another node or another new node, send them to the new node
//Reallocation of keys to handle permanent failure
func (node *Node) ScanDB(){
    fmt.Println("Entering ScanDB()")
    db := node.NodeDB
    markedForRemoval := []string{}
	err := db.View(func(txn *badger.Txn) error {
        opts := badger.DefaultIteratorOptions
        opts.PrefetchSize = 10
        it := txn.NewIterator(opts)
        defer it.Close()
        for it.Rewind(); it.Valid(); it.Next() {
	    item := it.Item()
	    k := item.Key()
        key := string(k)
        dstNodeHash,_,allocErr := node.Ring.AllocateKey(key)
        if allocErr != nil{
            fmt.Errorf("Failed to allocate key")
        }
        isCoordinator := contains(node.NodeRingPositions,dstNodeHash)

        if isCoordinator == false{
            var cartBytes []byte
            err := item.Value(func(val []byte) error {
            // This func with val would only be called if item.Value encounters no error.

            // Copying or parsing val is valid.
            cartBytes = append([]byte{}, val...)

            return nil
            })
            if err != nil{
                fmt.Errorf("Error in retrieving cartData from DB")
            }
	        fmt.Printf("[Node %s] Sending data for key=%s to Node %s\n",node.CName, k, node.Ring.RingNodeDataArray[dstNodeHash].ID)
            cartData := map[string][]byte{key:[]byte(cartBytes)}
            writeMsg := &Message{
                SenderIP:node.IP,SenderPort:node.Port,Data:cartData,
            }
            //fmt.Printf("ScanDB - dstNodeUrl is %s\n",dstNodeUrl)
            //fmt.Printf("ScanDB - dstNodeHash is ")
            rChannel := make(chan Message)
            go func(dstNodeHash int,msgChnl chan Message,msgToSend *Message) {
                //Check status of the dst node's physical node. If down, look for next best option 
                node.CheckStatusAndSend(dstNodeHash,msgChnl,msgToSend,"put")
            }(dstNodeHash,rChannel,writeMsg)
            //responseMessage := <-rChannel
            <-rChannel
            fmt.Printf("[Node %s] ScanDB() completes transfer \n",node.CName)
            //Check if this node is still inside the preference list, if not mark this key-value pair for removal from database
            if ( InPrefList(node.Ring.NodePrefList[dstNodeHash],node.IP,node.Port) == false){
                fmt.Printf("[Node %s] Removing key %s from database\n",node.CName,key)
                markedForRemoval = append(markedForRemoval,key)
            }
        }

	  }
	  return nil
	})
    handle(err)
    //Delete the keys that were marked for removal
    for _,staleKey := range markedForRemoval{
        deleteErr := node.DeleteKey(staleKey)
        if deleteErr != nil{
            fmt.Errorf("Failed to delete key [%s]\n",staleKey)
        }
    }
}

func (node *Node) GetNodeHandler(w http.ResponseWriter, r *http.Request) {
    var msg *Message

    w.Header().Set("Content-Type", "application/json")
    if r.Body == nil {
        http.Error(w, "Please send a request body", 400)
        return
    }

    err := json.NewDecoder(r.Body).Decode(&msg)
    if err != nil {
        http.Error(w, err.Error(), 400)
        return
    }

    ring := node.Ring
    query := msg.Query //Get key
    dstNodeHash, dstNodeUrl, AllocErr := ring.AllocateKey(query)
    if AllocErr != nil{
        fmt.Printf("Failed to allocate node to key [%s]\n",query)
    }
    responseData := make(map[string][]byte)
    responseData["key"]=[]byte(query)
    responseData["nodeId"]=[]byte(ring.RingNodeDataArray[dstNodeHash].ID)
    responseData["nodeUrl"]=[]byte(dstNodeUrl)
    responseMessage := &Message{
        SenderIP:node.IP,SenderPort:node.Port,Data:responseData,
    }
    fmt.Printf("[Node %s] Responding to Client's query\n",node.CName)
    json.NewEncoder(w).Encode(responseMessage)
}

func (node *Node) HeartbeatHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK) //Set response code to 200
    fmt.Fprintf(w,"") //Just send a blank reply at least the server knows you're reachable 
}



func (node *Node) HttpClientReq(msg *Message,targetUrl string,endpoint string, relayChannel chan Message){
	client := &http.Client{
	}
    //fmt.Println("HTTP Client Req function called")
    url := fmt.Sprintf("http://%s/%s",targetUrl,endpoint)
    fmt.Printf("[Node %s] Sending HTTP Req to url: %s\n",node.CName,url)
    jsonBuffer, err := json.Marshal(msg)
    handle(err)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBuffer))
    req.Header.Set("Content-Type", "application/json")

    res, err := client.Do(req)
    defer res.Body.Close()
    fmt.Println("HTTP Client Req - Got a response")

    // always close the response-body, even if content is not required
    if err != nil {
         fmt.Println("Unable to reach the server.")
    } else {
        if res.StatusCode == 200{
            var resMsg Message
            json.NewDecoder(res.Body).Decode(&resMsg)
            //TODO: Remove comments for statement below(Removing for hintedhandoff testing)
            //fmt.Printf("Response Message is \n%v\n",resMsg)
            relayChannel <- resMsg
        }else{
            fmt.Printf("res.StatusCode is %d\n",res.StatusCode)
        }
    }
}


//Adds item into the badger database
func (node *Node) UpdateDB(update map[string][]byte) error{
    fmt.Println("Updating database")
    db := node.NodeDB
    txn := db.NewTransaction(true)
    for k,v := range update{
      if err := txn.Set([]byte(k),[]byte(v)); err == badger.ErrTxnTooBig {
        _ = txn.Commit()
        txn = db.NewTransaction(true)
        _ = txn.Set([]byte(k),[]byte(v))
      }
    }
    err := txn.Commit()
    return err
}

//print all key, value pairs in the badger database
func (node *Node) ViewDB(){
    db := node.NodeDB
	err := db.View(func(txn *badger.Txn) error {
	  opts := badger.DefaultIteratorOptions
	  opts.PrefetchSize = 10
	  it := txn.NewIterator(opts)
	  defer it.Close()
	  for it.Rewind(); it.Valid(); it.Next() {
	    item := it.Item()
	    k := item.Key()
	    err := item.Value(func(v []byte) error {
	      fmt.Printf("key=%s, value=%s\n", k, v)
	      return nil
	    })
	    if err != nil {
	      return err
	    }
	  }
	  return nil
	})
    handle(err)
}
//Looks up the value associated to the key in the database
func (node *Node) QueryDB(queryKey string) (map[string][]byte,error){
	var outputVal []byte
    var valCopy []byte
    db := node.NodeDB
	err := db.View(func(txn *badger.Txn) error {
    item, err := txn.Get([]byte(queryKey))

    if err!=nil{
        glog.Error(err)
	    return err
    }

	err = item.Value(func(val []byte) error {
	// This func with val would only be called if item.Value encounters no error.

	// Copying or parsing val is valid.
	valCopy = append([]byte{}, val...)

	return nil
	})

    if err!=nil{
        glog.Error(err)
	    return err
    }

	// You must copy it to use it outside item.Value(...).
	fmt.Printf("Value for key [%s] is: %s\n",queryKey, valCopy)

	return nil
	})

    outputVal = valCopy
    output := make(map[string][]byte)
    output[queryKey]=outputVal
	return output, err
}

func (node *Node) DeleteKey(Key string) error{
    db := node.NodeDB
	err := db.Update(func(txn *badger.Txn) error {
	err := txn.Delete([]byte(Key))
	if err!=nil{
        return err
    }

	return nil
	})
    return err
}


func (node *Node) RunHintedHandOff(replicaHash int, userID string, clientCartBytes []byte){
    //Triggers when status of a physical node is down when a request is made(used during put requests)
    hashAndUserId := strconv.Itoa(replicaHash) + "," + userID

    hintedHandoff := map[string][]byte{hashAndUserId:clientCartBytes}
    node.UpdateHH(hintedHandoff)
}
func (node *Node) CheckHintedHandOff(){
    //Convert hintedhandoff database to a map[string][]byte
    hhMap := node.HHDBtoMap()
    if len(hhMap) == 0{
        // Don't check the hinted handoff database if it is empty
        
    }else{
        ring := node.Ring
        wChannel := make(chan Message)
        for hashandUserID, CartDataInHH := range(hhMap){
            //Split the key into replicanodehash and user id(key for shopping cart)
            arrayHashUserID := strings.Split(hashandUserID, ",")
            fmt.Printf("arrayHashUserID[1]: %s\n", arrayHashUserID[1])
            replicaHash,err := strconv.Atoi(arrayHashUserID[0])
            userID := arrayHashUserID[1]
            if err != nil{
                fmt.Println("Hash conversion failed")
            }else{
                replicaData:= ring.RingNodeDataArray[replicaHash]
                //Check if nodestatus is alive. If alive send If not skip
                //Assume not alive and then we send from here.
                fmt.Printf("Replica Node Data of Hinted Handoff: %v\n",replicaData)
                //Check if replica data exists in node.
                //If replcia data is empty, it means replica node has permanently failed and removed from ring
                //If not empty means it has only fainted
                if replicaData == (ConHash.NodeData{}){
                    fmt.Println("Replica has permanently failed. Proceed to remove related Hinted Handoff")
                    //Remove from hinted handoff db
                    node.DeleteHHKey(hashandUserID)
                }else{
                    physicalNodeID := replicaData.CName + "0"
                    statusOfPhysicalNode := node.Ring.NodeStatuses[physicalNodeID]
                    if statusOfPhysicalNode == true{
                        cartData := map[string][]byte{userID:CartDataInHH}
                        writeMsg := &Message{
                            SenderIP:node.IP,SenderPort:node.Port,Data:cartData,
                        }
                        replicaNodeDataUrl := fmt.Sprintf("%s:%s",replicaData.IP,replicaData.Port)
                        //Send hintedhandoff
                        go node.HttpClientReq(writeMsg,replicaNodeDataUrl,"put",wChannel)
                        select {
                        case <-wChannel:
                            fmt.Println("Hinted Handoff Success!")
                            //Remove item from hinted handoff DB (deleteKey with the hashandUserID)
                            node.DeleteHHKey(hashandUserID)
                        case <-time.After(time.Duration(5) * time.Second):
                            fmt.Println("Hinted Handoff Timeout!\n")
                            //Physical node did not respond even though alive
                            //Go to next item in hintedhandoff and don't remove current hinted handoff.
                        }
                    }else{
                        //Go to next item in hintedhandoff as the physical node is still down
                    }
                }


            }
        }
    }

}

//Converts the hinted handoff badger database into a keyvalue map
func (node *Node) HHDBtoMap() map[string][]byte {
    db := node.HHQueue
    hhQueue := make(map[string][]byte)
    err := db.View(func(txn *badger.Txn) error {
        opts := badger.DefaultIteratorOptions
        opts.PrefetchSize = 10
        it := txn.NewIterator(opts)
        defer it.Close()
        for it.Rewind(); it.Valid(); it.Next() {
        item := it.Item()
        k := item.Key()
        err := item.Value(func(v []byte) error {
            //fmt.Printf("key=%s, value=%s\n", k, v)
            hhQueue[string(k)] = v
            return nil
        })
        if err != nil {
            return err
        }
        }
        return nil
    })
    handle(err)
    return hhQueue
}

//Updates the hinted handoff badger database by adding new hinted handoff
func (node *Node) UpdateHH(update map[string][]byte) error{
    db := node.HHQueue
    txn := db.NewTransaction(true)
    for k,v := range update{
      if err := txn.Set([]byte(k),[]byte(v)); err == badger.ErrTxnTooBig {
        _ = txn.Commit()
        txn = db.NewTransaction(true)
        _ = txn.Set([]byte(k),[]byte(v))
      }
    }
    err := txn.Commit()
    return err
}

//Deletes hintedhandoff item in hinted handoff badger database
func (node *Node) DeleteHHKey(Key string) error{
    db := node.HHQueue
	err := db.Update(func(txn *badger.Txn) error {
	err := txn.Delete([]byte(Key))
	if err!=nil{
        return err
    }

	return nil
	})
    return err
}

func (node *Node) CheckStatusAndSend(dstNodeHash int, msgChnl chan Message, msg *Message, endpoint string){
    //Takes dstNodeHash , response message channel and message as argument
    //Function checks if dst node's physical node is alive. If alive, send. If not find next alive from pref list. 
    dstNodeData := node.Ring.RingNodeDataArray[dstNodeHash]
    
    dstPhysicalNodeID := dstNodeData.CName + "0"
    dstNodeURL := fmt.Sprintf("%s:%s",dstNodeData.IP,dstNodeData.Port)
    statusOfdstPhysicalNode := node.Ring.NodeStatuses[dstPhysicalNodeID]
    if statusOfdstPhysicalNode == false{
        //Look for next alive physical node and send the message to it
        fmt.Println("Physical Node of Coordinator Node is down")
        fmt.Printf("node.Ring.NodePrefList[dstNodeHash]: %v\n",node.Ring.NodePrefList[dstNodeHash])
        for _, nodeData := range node.Ring.NodePrefList[dstNodeHash]{
            //Look through preference list for the next alive physical node to send to
            physicalNodeID := nodeData.CName + "0"
            statusOfPhysicalNode := node.Ring.NodeStatuses[physicalNodeID]
            if statusOfPhysicalNode == true{
                fmt.Printf("Relaying to %v instead\n",nodeData)
                newDstNodeURL := fmt.Sprintf("%s:%s",nodeData.IP,nodeData.Port)
                node.HttpClientReq(msg,newDstNodeURL,endpoint,msgChnl)
                break
            }
        }
    }else{
        //Physical node of coordinator node is alive
        fmt.Println("Sending to Physical Node of Coordinator Node ")
        node.HttpClientReq(msg,dstNodeURL,endpoint,msgChnl)
    }
}

func (node *Node) runCommand(commandStr string) error {
    // To-Do : Add a command to view node's attributes and variables
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr, parseErr := Utils.ParseCommandLine(commandStr)
	handle(parseErr)

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

help: Shows lists of commands

exit: quits program

query: Usage - query <key>
query searches the database for a key and returns the value

update: Usage - update <key> <value>
update takes user inputted key value pair and updates the database

view: Views database

delete: Usage - delete <key>
delete deletes an existing key and its respective value from database

allocateKey: Usage - allocateKey <Key>
allocateKey searches the node's consistent hash ring and find out which node is
the coordinator node which is responsible for the read/write for the given key

httpPut: Usage - httpPut <targetIP> <targetPort> <key> <value>
httpPut sends user inputted data to another server and updates their database

httpGet: Usage - httpGet <targetIP> <targetPort> <key>
httpGet sends a key to another server, the receiving server will query its database
for the key-value pair and it in the response

httpGetNode: Usage - httpGetNode <targetIP> <targetPort> <key>
httpGetNode sends a key to another server, the receiving server will refer to the consistent
hash ring and find out which coordinator node is responsible for the read/writes of this key.
Then, the server will return a message to the client containing the Node ID and the Node's IP
address and port

`)
        case "query":
            if len(arrCommandStr)!=2{
                return fmt.Errorf("Usage of query - query <Key>")
            }
            key := arrCommandStr[1]
            fmt.Printf("Querying db...\n")
            results,err := node.QueryDB(key)
            if err!=nil{
                fmt.Printf("Key <%s> not found in datbase\n",key)
            } else{
            fmt.Printf("Query results are %s\n",results)
            }
        case "update":
           if len(arrCommandStr)!=3{
               return fmt.Errorf("Usage of update - update <key> <value>")
           }
           key := arrCommandStr[1]
           rawValue := arrCommandStr[2]
           value, marshalErr := json.Marshal(rawValue)
           handle(marshalErr)
           update := map[string][]byte{key:value}
           node.UpdateDB(update)
        case "view":
            if len(arrCommandStr)!=1{
                return fmt.Errorf("Extra arguments, usage of view - view")
            }
            node.ViewDB()
        case "delete":
            if len(arrCommandStr)!=2{
                return fmt.Errorf("Usage of delete - delete <Key>")
            }
            key := arrCommandStr[1]
            err := node.DeleteKey(key)
            if err!=nil{
                fmt.Printf("Key <%s> not in database, can't delete\n",key)
            } else {
                fmt.Printf("Key <%s> has been deleted from database\n",key)
            }
        case "allocateKey":
            if len(arrCommandStr)!=2{
                return fmt.Errorf("Usage of allocateKey - allocateKey <Key>")
            }
            key := arrCommandStr[1]
            dstNodeHash, dstNodeUrl, AllocErr := node.Ring.AllocateKey(key)
            if AllocErr!=nil{
                fmt.Printf("Failed to allocate key to a Coordinator Node\n",key)
            } else {
                fmt.Printf("Coordinator for key <%s> is Node %s and its url is %s\n",key,node.Ring.RingNodeDataArray[dstNodeHash].ID ,dstNodeUrl)
            }
		case "httpPut":
			//Do nothing
			if len(arrCommandStr)!=5{
				return fmt.Errorf("Usage of httpPut - httpPut <targetIP> <targetPort> <key> <value")
			}
			httpMsg := &Message{}
			httpMsg.SenderIP = node.IP
			httpMsg.SenderPort = node.Port
            key := arrCommandStr[3]
            rawValue := arrCommandStr[4]
            value, marshalErr := json.Marshal(rawValue)
            handle(marshalErr)
            data := map[string][]byte{key:value}
			httpMsg.Data = data
            fmt.Printf("httpMsg %s\n",httpMsg)
            targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
            rChannel := make(chan Message)
            node.HttpClientReq(httpMsg,targetUrl,"put",rChannel)
            <-rChannel
            close(rChannel)
        case "httpGet":
            if len(arrCommandStr)!=4{
                return fmt.Errorf("Usage of httpGet - httpGet <targetIP> <targetPort> <key to query>")
            }
            httpMsg := &Message{}
            httpMsg.SenderIP = node.IP
            httpMsg.SenderPort = node.Port
            key := arrCommandStr[3]
            httpMsg.Query = key
            fmt.Printf("httpMsg %s\n",httpMsg)
            targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
            rChannel := make(chan Message)
            node.HttpClientReq(httpMsg,targetUrl,"get",rChannel)
            <-rChannel
            close(rChannel)
        case "httpGetNode":
            if len(arrCommandStr)!=4{
                return fmt.Errorf("Usage of httpGetNode - httpGetNode <targetIP> <targetPort> <key to query>")
            }
            httpMsg := &Message{}
            httpMsg.SenderIP = node.IP
            httpMsg.SenderPort = node.Port
            key := arrCommandStr[3]
            httpMsg.Query = key
            fmt.Printf("httpMsg %s\n",httpMsg)
            targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
            rChannel := make(chan Message)
            node.HttpClientReq(httpMsg,targetUrl,"get-node",rChannel)
            <-rChannel
            close(rChannel)
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

    if len(os.Args) != 5{
        fmt.Printf("Usage of program is: %s <PORT> <DBPath> <NodeNumID> <bool: If node should " +
            "register with server, False to simulate revival>\n", os.Args[0])
        os.Exit(0)
    }
	//Set constants here
	const NUMBER_OF_VNODES = config.NUMBER_OF_VNODES;
	const MAX_KEY = config.MAX_KEY;
    const REPLICATION_FACTOR = config.REPLICATION_FACTOR;
    const RW_FACTOR = config.RW_FACTOR;

    currentIP, err := Utils.ExternalIP()
    fmt.Printf("Setting Node's IP to be %s\n",currentIP)
    handle(err)
    port := os.Args[1]
    DBPath := os.Args[2]

    NodeNumID,strconverr := strconv.Atoi(os.Args[3])

    shouldRegisterWithServer, strconverr := strconv.ParseBool(os.Args[4])
    if strconverr != nil{
        log.Fatalln(strconverr)
    }


    if strconverr != nil{
        fmt.Errorf("Failed to convert NodeNumID to int. Please enter an integer")
    }
    //fmt.Println("Testing 1")


	conNode := ConHash.NewNode(NodeNumID, NUMBER_OF_VNODES,DBPath,currentIP,port, &ConHash.Ring{
        MaxID:             0,
        RingNodeDataArray: nil,
        NodePrefList:      nil,
        ReplicationFactor: 0,
        RWFactor:          0,
        NodeStatuses:      nil,
    })
    node := Node{conNode}
    //TODO recovery for ring
    oldRingPath := fmt.Sprintf("/tmp/%s_RING",node.CName)
    ringInMemory := Utils.FileExists(oldRingPath)
    if ringInMemory{
        fmt.Printf("[Node %s] Reloading ring from memory\n",node.CName)
        ringJson, readFileErr := ioutil.ReadFile(oldRingPath)
        if readFileErr != nil{
            fmt.Errorf("Error trying to read Ring from JSON file")
        }
        var oldRing ConHash.Ring
        unmarshalErr := json.Unmarshal(ringJson, &oldRing)
        if unmarshalErr != nil{
            fmt.Errorf("Failed to unmarshal content from json file into ring")
        }
        node.Ring = &oldRing
    }

    const REPLICATIONFACTOR = config.REPLICATION_FACTOR

    go node.Start()
    time.Sleep(time.Duration(WARMUP_DURATION) * time.Second)
    //Important to put registration after start. So that if the server fails early,
    //it should not register
    if shouldRegisterWithServer {
        node.RegisterWithRingServer("http://" + ConHash.RING_URL)
    }

	//Start of CLI interactivity
	reader := bufio.NewReader(os.Stdin)
    fmt.Printf("Node %s@%s:%s$ ",node.CName,node.IP,node.Port)
	for {
        fmt.Printf("Node %s@%s:%s$ ",node.CName,node.IP,node.Port)
		cmdString, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		err = node.runCommand(cmdString)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

