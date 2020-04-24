package main

import (
    "50.041-DistSysProject-BunshinDB/pkg/Utils"
    "50.041-DistSysProject-BunshinDB/pkg/ConHash"
    "50.041-DistSysProject-BunshinDB/pkg/ShoppingCart"
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
)


type Message struct{
    SenderIP string
    SenderPort string
    Data map[string][]byte //Key-Value pair
    Query string //Just a key string for receiver to query
    //ResponseCode string //200,404 etc.
}

type Node struct{
    ConHash.Node
    //ResponseChannel chan interface{}
    //TimeoutChannel chan interface{} 
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
    fmt.Println(msg)
    query := msg.Query
    ring := node.Ring
    dstNodeHash, dstNodeUrl , AllocErr := ring.AllocateKey(query)
    if AllocErr != nil{
        fmt.Println("Failed to allocate node to key [%s]",query)
    }

    //TODO Allow Nodes to check if they're in the Coordinator Node's pref list
    //If so, let them retrieve the item from their database
    if (contains(node.NodeRingPositions,dstNodeHash) || InPrefList(ring.NodePrefList[dstNodeHash],node.IP,node.Port)){ //If this node is responsible 
        fmt.Println("This Node is Coordinator or inPrefList")

        //First, try to retrieve the data from the databaseFirst, try to retrieve the data from the database
        fmt.Println("Get Handler - Retrieving Key Value pair and sending it back to Requestor")
        var responseStatus string
        queryResponse, err := node.QueryDB(query)
        if err != nil{
            responseStatus = "404"
        } else {
            responseStatus = "200"
        }
        //TODO Need to Implement R mechanism here
        if ((ring.RingNodeDataArray[dstNodeHash].IP == msg.SenderIP && ring.RingNodeDataArray[dstNodeHash].Port == msg.SenderPort) ||
        InPrefList(ring.NodePrefList[dstNodeHash],msg.SenderIP,msg.SenderPort)){
            //If this request is due to the R process
            fmt.Println("This Node is responding to a R broadcast")
            responseMessage := &Message{
                SenderIP:node.IP,SenderPort:node.Port,Data:queryResponse,
            }
            if responseStatus == "404"{
                http.Error(w,http.StatusText(http.StatusNotFound),http.StatusNotFound)
            }
            json.NewEncoder(w).Encode(responseMessage)
        } else{
            //This node has to take initiative to start the R process.
            fmt.Println("This Node is initiating an R broadcast")
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
                    go func(rData ConHash.NodeData){
                        replicaNodeDataUrl := fmt.Sprintf("%s:%s",rData.IP,rData.Port)
                        fmt.Printf("Debugging replica url %s\n",replicaNodeDataUrl)
                        node.HttpClientReq(rMessage,replicaNodeDataUrl,"get",rChannel)
                        responseMessage := <-rChannel
                        if len(shoppingCartVersions) < ring.RWFactor{
                            //Convert bytes in data to shopping cart structure
                            data := responseMessage.Data
                            shoppingCartBytes := data[msg.Query]
                            var shoppingCart ShoppingCart.ShoppingCart
                            json.Unmarshal(shoppingCartBytes,&shoppingCart)
                            //json.NewDecoder(shoppingCartBytes).Decode(&shoppingCart)
                            shoppingCartVersions = append(shoppingCartVersions,shoppingCart)
                        } else{
                            // Do nothing :')
                        }
                    }(replicaNodeData)
                }
            }
            //close(rChannel)
            //Reconcile differences
            listOfConflictingShoppingCarts := ShoppingCart.CompareShoppingCarts(shoppingCartVersions)
            //If reconciliation is successful
            if len(listOfConflictingShoppingCarts)== 1{
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
            //queryResponse,err :=
            } else if len(listOfConflictingShoppingCarts) > 1{
                //There are multiple conflicting versions of the shopping cart
                //Need to let the Client know somehow maybe with multiple key val pairs?
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
        fmt.Println("Get Handler - Relaying Key to the Coordinator Node")
        // TODO Implement a fallback mechanism if Coordinator Node is not alive

        //Need to relay get request to appropriate node
        rChannel := make(chan Message)
        node.HttpClientReq(msg,dstNodeUrl,"get",rChannel)
        fmt.Println("Get Handler - Returning relayed message to client")
        responseMessage := <-rChannel
        fmt.Println("Received Relayed Msg from Coordinator Node")
        close(rChannel)
        json.NewEncoder(w).Encode(responseMessage)
    }
}


func (node *Node) PutHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Println("Put Handler activated")
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
    fmt.Println("Put Handler - Allocating Key")
    for key, _ := range msgData{
        dstNodeHash, dstNodeUrl , AllocErr := ring.AllocateKey(key) //Get the destination node of this key
        if AllocErr != nil{
            fmt.Println("Failed to allocate node to key [%s]",key)
            http.Error(w, err.Error(), 400)
            return
        }
        //TODO Allow Nodes to check if they're in the Coordinator Node's pref list
        //If so, let them retrieve the item from their database
        if (contains(node.NodeRingPositions,dstNodeHash) || InPrefList(ring.NodePrefList[dstNodeHash],node.IP,node.Port)){ //If this node is responsible 
            fmt.Println("Node is Coordinator/InPrefList")

            //First, try to retrieve the data from the databaseFirst, try to retrieve the data from the database
            //fmt.Println("Get Handler - Retrieving Key Value pair and sending it back to Requestor")
            //TODO Need to Implement W mechanism here
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
                    json.NewEncoder(w).Encode(responseMessage)
                }
            } else{
                fmt.Println("Node is initiating W process")
                fmt.Println("This is because the msg was sent by %s:%s\n",msg.SenderIP,msg.SenderPort)
                //This node has to take initiative to start the W process.
                //var responseStatus string
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

                for _,replicaNodeData := range otherReplicas{
                    if replicaNodeData.CName != node.CName{
                        //Need to pay attention to this when debugging
                        go func(rData ConHash.NodeData, replicationPointer *int) {
                            replicaNodeDataUrl := fmt.Sprintf("%s:%s",rData.IP,rData.Port)
                            node.HttpClientReq(writeMsg,replicaNodeDataUrl,"put",wChannel)
                            <-wChannel
                            *replicationPointer = *replicationPointer + 1
                        }(replicaNodeData,repPointer)
                    }
                }
                //close(wChannel)
                if successfulReplications >= ring.RWFactor{
                    //Write is successful
                    responseMessage := &Message{
                        SenderIP:node.IP,SenderPort:node.Port,Data:cartData,
                    }
                    w.WriteHeader(http.StatusOK)
                    json.NewEncoder(w).Encode(responseMessage)
                } else{
                    //Return 501 code because Server failed to complete write (which means alot of failures in DB)
                    http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
                }
            }
        } else{
            fmt.Println("Node is relaying client request to Coordinator")
            //Need to relay put request to appropriate node

            //TODO In case appropriate node fails, check pref list and send to secondary

            //dstNodeData := ring.RingNodeDataArray[dstNodeHash]
            //dstNodeIPPort := dstNodeUrl
            rChannel := make(chan Message)

            node.HttpClientReq(msg,dstNodeUrl,"put",rChannel)
            relayResponseMsg := <-rChannel
            close(rChannel)
            json.NewEncoder(w).Encode(relayResponseMsg)
        }
    }
}

func (node *Node) NewRingHandler(w http.ResponseWriter, r *http.Request) {
    //TODO update ring
    //Need a onUpdateRing function in conHash.go

    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        log.Fatalln(err)
    }

    var ring ConHash.Ring
    err = json.Unmarshal(body, &ring)

    if err != nil {
        log.Println(err)
    }

    node.Ring = &ring
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
        fmt.Println("Failed to allocate node to key [%s]",query)
    }
    responseData := make(map[string][]byte)
    responseData["key"]=[]byte(query)
    responseData["nodeId"]=[]byte(ring.RingNodeDataArray[dstNodeHash].ID)
    responseData["nodeUrl"]=[]byte(dstNodeUrl)
    responseMessage := &Message{
        SenderIP:node.IP,SenderPort:node.Port,Data:responseData,
    }
    fmt.Println(responseMessage)
    json.NewEncoder(w).Encode(responseMessage)
}

func (node *Node) HeartbeatHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK) //Set response code to 200
    fmt.Fprintf(w,"") //Just send a blank reply at least the server knows you're reachable 
}



//Think of passing in a channel as an argument to prevent potential
//mixed usage of the same channel
func (node *Node) HttpClientReq(msg *Message,targetUrl string,endpoint string, relayChannel chan Message){
	client := &http.Client{
	}
    fmt.Println("HTTP Client Req function called")
    url := fmt.Sprintf("http://%s/%s",targetUrl,endpoint)

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
            fmt.Printf("Response Message is \n%v\n",resMsg)
                relayChannel <- resMsg
        }
    }
}



func (node *Node) UpdateDB(update map[string][]byte) error{
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

//print all key, value pairs
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

	//var valCopy []byte
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
	fmt.Printf("The answer is: %s\n", valCopy)

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

    if len(os.Args) != 4{
        fmt.Printf("Usage of program is: %s <PORT> <DBPath> <NodeNumID>\n", os.Args[0])
        os.Exit(0)
    }
	//Set constants here
	const NUMBER_OF_VNODES = 4;
	const MAX_KEY = 100;
    const REPLICATION_FACTOR = 3;
    const RW_FACTOR = 1;

    currentIP, err := Utils.ExternalIP()
    fmt.Printf("Setting Node's IP to be %s\n",currentIP)
    //TODO REMOVE THIS ONCE LIONEL'S RING SERVER FUNCTION WORKS
    currentIP = "127.0.0.1"
    handle(err)
    port := os.Args[1]
    DBPath := os.Args[2]
    NodeNumID,strconverr := strconv.Atoi(os.Args[3])

    if strconverr != nil{
        fmt.Errorf("Failed to convert NodeNumID to int. Please enter an integer")
    }
    fmt.Println("Testing 1")

    ring := ConHash.NewRing(MAX_KEY,REPLICATION_FACTOR,RW_FACTOR)
	conNode := ConHash.NewNode(NodeNumID, NUMBER_OF_VNODES,DBPath,currentIP,port,ring)
    node := Node{conNode}
	//should with assign the ring to node.ring only when we register with ring?
	//node.RegisterWithRing(node.Ring)
    //For demo purposes, gonna hard code a ring
    //const MAXID = 100
    const REPLICATIONFACTOR = 3
    NodeDataArray := make([]ConHash.NodeData,MAX_KEY,MAX_KEY)
    fmt.Println("Hello")

    NodeDataArray[4] = ConHash.NodeData{"A0","A",4,"127.0.0.1","8080"}
    NodeDataArray[9] = ConHash.NodeData{"B0","B",9,"127.0.0.1","8081"}
    NodeDataArray[14] = ConHash.NodeData{"C0","C",14,"127.0.0.1","8082"}
    NodeDataArray[19] = ConHash.NodeData{"D0","D",19,"127.0.0.1","8083"}
    NodeDataArray[24] = ConHash.NodeData{"A1","A",24,"127.0.0.1","8080"}
    NodeDataArray[29] = ConHash.NodeData{"B1","B",29,"127.0.0.1","8081"}
    NodeDataArray[34] = ConHash.NodeData{"C1","C",34,"127.0.0.1","8082"}
    NodeDataArray[39] = ConHash.NodeData{"D1","D",39,"127.0.0.1","8083"}
    NodeDataArray[44] = ConHash.NodeData{"A2","A",44,"127.0.0.1","8080"}
    NodeDataArray[49] = ConHash.NodeData{"B2","B",49,"127.0.0.1","8081"}
    NodeDataArray[54] = ConHash.NodeData{"C2","C",54,"127.0.0.1","8082"}
    NodeDataArray[59] = ConHash.NodeData{"D2","D",59,"127.0.0.1","8083"}
    NodeDataArray[64] = ConHash.NodeData{"A3","A",64,"127.0.0.1","8080"}
    NodeDataArray[69] = ConHash.NodeData{"B3","B",69,"127.0.0.1","8081"}
    NodeDataArray[74] = ConHash.NodeData{"C3","C",74,"127.0.0.1","8082"}
    NodeDataArray[79] = ConHash.NodeData{"D3","D",79,"127.0.0.1","8083"}
    NodeDataArray[84] = ConHash.NodeData{"A4","A",84,"127.0.0.1","8080"}
    NodeDataArray[89] = ConHash.NodeData{"B4","B",89,"127.0.0.1","8081"}
    NodeDataArray[94] = ConHash.NodeData{"C4","C",94,"127.0.0.1","8082"}
    NodeDataArray[99] = ConHash.NodeData{"D4","D",99,"127.0.0.1","8083"}

    demoRing := &ConHash.Ring{
        MaxID: MAX_KEY,
        RingNodeDataArray:NodeDataArray,
        //NodePrefList:NodePrefList,
        NodePrefList: map[int][]ConHash.NodeData{},
        ReplicationFactor: REPLICATIONFACTOR,
    }

    demoRing.GenPrefList()

    fmt.Printf("Reloading Ring from memory: Ring is %v\n",demoRing)

    fmt.Printf("Nodes Preference Lists are: %v\n",demoRing.NodePrefList)

    node.Ring = demoRing
    for _,nodeData := range node.Ring.RingNodeDataArray{
        if nodeData.CName == node.CName{
            node.NodeRingPositions = append(node.NodeRingPositions,nodeData.Hash)
        }
    }
    fmt.Println(node.NodeRingPositions)

    nodeQuery := "A2"
	nodeIP, err := demoRing.GetNode(nodeQuery)
	if err == nil {
        fmt.Printf("Node %s found at : %s \n",nodeQuery,nodeIP)
    } else{
        fmt.Printf("Node %s not found\n",nodeQuery)
    }

    searchKey := "testing"
    nodeHash, addr, err := demoRing.AllocateKey(searchKey)
    if err == nil {
		fmt.Printf("Key [%s] found at node %s with ip [%s] \n",searchKey, demoRing.RingNodeDataArray[nodeHash].ID,addr)
	} else {
		fmt.Printf("Node for key [%s] not found \n", searchKey )
	}


    go node.Start()

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

