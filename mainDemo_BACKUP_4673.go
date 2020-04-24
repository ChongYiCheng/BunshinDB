package main

<<<<<<< HEAD
import (
	"50.041-DistSysProject-BunshinDB/Stetho"
	"encoding/json"
	"io/ioutil"
	"log"
    "fmt"
    badger "github.com/dgraph-io/badger"
    "net/http"
    "os"
    "os/exec"
    "strings"
    //"bufio"
    "bytes"
    "net"
    "errors"
    glog "github.com/golang/glog"
    "strconv"
<<<<<<< HEAD:conBadgerHTTPdemo.go
    "./ConHash"
    //"time"
=======
    "50.041-DistSysProject-BunshinDB/pkg/ConHash"
	"time"
>>>>>>> 08ac7f8d45135623cb19d547eb401ba25dab2b33:mainDemo.go
)


type Message struct{
    SenderIP string
    SenderPort string
    MessageType int
    Data map[string][]byte //Key-Value pair
    Query string //Just a key string for receiver to query
    ResponseCode string //200,404 etc.
    Timestamp []int //Vector Clock
}

type Node struct{
    ConHash.Node
    ResponseChannel chan interface{}
    TimeoutChannel chan interface{}
}

type Ring struct{
    ConHash.Ring
    stethoUrl string
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


func (node *Node) HttpServerStart(){

	http.HandleFunc("/get", node.GetHandler)
	http.HandleFunc("/put", node.PutHandler)
	http.HandleFunc("/new-ring", node.NewRingHandler)
	http.HandleFunc("/get-node", node.GetNodeHandler)
    http.HandleFunc("/hb", node.HeartbeatHandler)
    //Send hintedhandoff to right node
    //http.HandleFunc("/hh-send", node.HHHandler)
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

    if contains(node.NodeRingPositions,dstNodeHash){ //If this node is responsible 
        fmt.Println("Get Handler - Retrieving Key Value pair and sending it back to client")
        var responseStatus string
        queryResponse, err := node.QueryDB(query)
        if err != nil{
            responseStatus = "404"
        } else {
            responseStatus = "200"
        }
        responseMessage := &Message{
            SenderIP:node.IP,SenderPort:node.Port,Data:queryResponse,
            ResponseCode:responseStatus,Timestamp:[]int{},
        }
        json.NewEncoder(w).Encode(responseMessage)
    } else{
        fmt.Println("Get Handler - Relaying Key to the Coordinator Node")
        //Need to relay get request to appropriate node
		//dstNodeData := ring.RingNodeDataArray[dstNodeHash]
        //dstNodeIPPort := fmt.Sprintf("%s:%s",dstNodeData.IP,dstNodeData.Port)
        node.HttpClientReq(msg,dstNodeUrl,"get")
        fmt.Println("Get Handler - Returning relayed message to client")
        responseMessage := <-node.ResponseChannel
        fmt.Println("Received Relayed Msg from Coordinator Node")
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
        }
        if contains(node.NodeRingPositions,dstNodeHash){ //If this node is responsible 
            fmt.Println("Put Handler - Updating Database with Key Value pair")
            var responseStatus string
            err := node.UpdateDB(msgData)
            if err != nil{
                responseStatus = "404"
            } else {
                responseStatus = "200"
            }
            responseMessage := &Message{
                SenderIP:node.IP,SenderPort:node.Port,Data:msgData,
                ResponseCode:responseStatus,Timestamp:[]int{},
            }
            json.NewEncoder(w).Encode(responseMessage)
        } else{
            //Need to relay put request to appropriate node
            //dstNodeData := ring.RingNodeDataArray[dstNodeHash]
            //dstNodeIPPort := dstNodeUrl
            fmt.Println("Relaying msg to appropriate node")
            node.HttpClientReq(msg,dstNodeUrl,"put")
            relayResponse := <-node.ResponseChannel
            relayResponseMsg := relayResponse.(Message)
            json.NewEncoder(w).Encode(relayResponseMsg)
        }
    }
}

func (node *Node) NewRingHandler(w http.ResponseWriter, r *http.Request) {
    //To-Do update ring
    //Need a onUpdateRing function in conHash.go
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
        ResponseCode:"200",Timestamp:[]int{},
    }
    fmt.Println(responseMessage)
    json.NewEncoder(w).Encode(responseMessage)
}

func (node *Node) HeartbeatHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK) //Set response code to 200
    fmt.Fprintf(w,"") //Just send a blank reply at least the server knows you're reachable 
}
func (node *Node) HHHandler(w http.ResponseWriter, r *http.Request) {
    //Hinted Handoff Handler
    fmt.Println("HintedHandoff Handler activated")
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
    fmt.Println("HH Handler - Allocating Key")
    for key, _ := range msgData{
        dstNodeHash, dstNodeUrl , AllocErr := ring.AllocateKey(key) //Get the destination node of this key
        if AllocErr != nil{
            fmt.Println("Failed to allocate node to key [%s]",key)
        }
        if contains(node.NodeRingPositions,dstNodeHash){ //If this node is responsible 
            fmt.Println("HintedHandoff Handler - Updating Database with Key Value pair")
            var responseStatus string
            err := node.UpdateDB(msgData)
            if err != nil{
                responseStatus = "404"
            } else {
                responseStatus = "200"
            }
            responseMessage := &Message{
                SenderIP:node.IP,SenderPort:node.Port,Data:msgData,
                ResponseCode:responseStatus,Timestamp:[]int{},
            }
            json.NewEncoder(w).Encode(responseMessage)
        } else{
            //Need to relay put request to appropriate node
            //dstNodeData := ring.RingNodeDataArray[dstNodeHash]
            //dstNodeIPPort := dstNodeUrl
            fmt.Println("Hintedhandoff - Relaying msg to appropriate node")
            node.HttpClientReq(msg,dstNodeUrl,"hh-send")
            relayResponse := <-node.ResponseChannel
            relayResponseMsg := relayResponse.(Message)
            json.NewEncoder(w).Encode(relayResponseMsg)
        }
    }
} 
func (node *Node) handleMessage(m *Message) *Message{
    switch m.MessageType{
    case 0:
        //If Message type is GET
        var responseStatus string
        query := m.Query
        queryResponse, err := node.QueryDB(query)
        if err != nil{
            responseStatus = "404"
        } else {
            responseStatus = "200"
        }
        responseMessage := &Message{
            SenderIP:node.IP,SenderPort:node.Port,Data:queryResponse,
            ResponseCode:responseStatus,Timestamp:[]int{},
        }
        return responseMessage
    case 1:
        // If Message is to PUT
        data := m.Data
        node.UpdateDB(data)
        responseMessage := &Message{
            SenderIP:node.IP,SenderPort:node.Port,
            ResponseCode:"200",Timestamp:[]int{},
        }
        return responseMessage
    }
    return nil
}

func (node *Node) HttpClientReq(msg *Message,targetUrl string,endpoint string){
	client := &http.Client{
	}
    fmt.Println("HTTP Client Req function called")
    url := fmt.Sprintf("http://%s/%s",targetUrl,endpoint)

    jsonBuffer, err := json.Marshal(msg)
    handle(err)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBuffer))
    req.Header.Set("Content-Type", "application/json")

    fmt.Printf("Message: %v, targetURL: %s, endpoint: %s\n",msg,targetUrl,endpoint)
    res, err := client.Do(req)
    //defer res.Body.Close()
    fmt.Println(err)
    // always close the response-body, even if content is not required
    if err != nil {
        fmt.Println("Unable to reach the server.")
        if endpoint == "put"{
            fmt.Println("Commencing hinted handoff")
            node.UpdateHH(msg.Data)
            node.CheckHH()
            //node.ViewHH()
        }

    }else{
        fmt.Println("HTTP Client Req - Got a response")
        defer res.Body.Close()
        var resMsg Message
		json.NewDecoder(res.Body).Decode(&resMsg)
        fmt.Printf("Response Message is \n%v\n",resMsg)
        go func(){
            node.ResponseChannel <- resMsg
        }()
        //select{
        //case node.ResponseChannel <- resMsg:
        //    fmt.Println("Hello hello my name's dibo")
        //        // Do nothing
        //case <- time.After(100 * time.Millisecond):
        //    fmt.Println("Hello hello my name's nobo")
        //    //Do nothing
        //}
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

func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}
func (node *Node) CheckHH(){
    //Think about how to check through the hinted handoff db
    hhMap := node.HHDBtoMap()
    ring := node.Ring
    //Iterate through hhmap
    for key, _ := range hhMap{
        fmt.Printf("HH Key: %s\n",key)
        fmt.Println("HH Handler - Allocating Key")
        dstNodeHash, dstNodeUrl , AllocErr := ring.AllocateKey(key) //Get the destination node of this key
        if AllocErr != nil{
            fmt.Println("Failed to allocate node to key [%s]",key)
        }
        if contains(node.NodeRingPositions,dstNodeHash){ //If this node is responsible 
            fmt.Println("HintedHandoff Handler - Updating Database with Key Value pair")
            fmt.Printf("Key: %s, Value %s\n",key,hhMap[key])
            // var responseStatus string
            fmt.Printf("Key: %s, Value %s\n",key,hhMap[key])
            update := map[string][]byte{key:hhMap[key]}
            err := node.UpdateDB(update)
            if err != nil{
                //error
                fmt.Println("Error: Unable to update database due to %v\n",err) 
            }else{
                fmt.Println("Success in putting hintedhandoff - Proceed to remove from HHDB")
                node.DeleteHHKey(key)
            // }
            // responseMessage := &Message{
            //     SenderIP:node.IP,SenderPort:node.Port,Data:msgData,
            //     ResponseCode:responseStatus,Timestamp:[]int{},
            // }
            // json.NewEncoder(w).Encode(responseMessage)
            }
        }else{
            //Need to relay put request to appropriate node
            //dstNodeData := ring.RingNodeDataArray[dstNodeHash]
            //dstNodeIPPort := dstNodeUrl
            fmt.Printf("Hintedhandoff - Relaying msg to appropriate node: %s\n",dstNodeUrl)
            // dstNodeURLSplit := strings.Split(dstNodeUrl,":")
            // destIP := dstNodeURLSplit[0]
            // destPort := dstNodeURLSplit[1]
            fmt.Printf("Key: %s, Value %s\n",key,hhMap[key])
            update := map[string][]byte{key:value}
            fmt.Printf("KeyValuePair to be relayed for HH: %v\n",update)
            //node.UpdateDB(update)
            // node.HttpClientReq(msg,dstNodeUrl,"hh-send")
            // relayResponse := <-node.ResponseChannel
            // relayResponseMsg := relayResponse.(Message)
            // json.NewEncoder(w).Encode(relayResponseMsg)
        }
        
    }
    // httpMsg := &Message{}
    // httpMsg.SenderIP = node.IP
    // httpMsg.SenderPort = node.Port
    // httpMsg.MessageType = 1
    // key := arrCommandStr[3]
    // rawValue := arrCommandStr[4]
    // value, marshalErr := json.Marshal(rawValue)
    // handle(marshalErr)
    // data := map[string][]byte{key:value}
    // httpMsg.Data = data
    // fmt.Printf("httpMsg %s\n",httpMsg)
    // targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
    // node.HttpClientReq(httpMsg,targetUrl,"put")
}

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

func (node *Node) ViewHH(){
    db := node.HHQueue
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


func (node *Node) QueryHH(queryKey string) (map[string][]byte,error){
	var outputVal []byte
    var valCopy []byte
    db := node.HHQueue
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

func parseCommandLine(command string) ([]string, error) {
	//Finite state machine to handle arguments with white spaces enclosed within quotes
	//Handles escaped stuff too
    var args []string
    state := "start"
    current := ""
    quote := "\""
    escapeNext := true
    for i := 0; i < len(command); i++ {
        c := command[i]

        if state == "quotes" {
            if string(c) != quote {
                current += string(c)
            } else {
                args = append(args, current)
                current = ""
                state = "start"
            }
            continue
        }

        if (escapeNext) {
            current += string(c)
            escapeNext = false
            continue
        }

        if (c == '\\') {
            escapeNext = true
            continue
        }

        if c == '"' || c == '\'' {
            state = "quotes"
            quote = string(c)
            continue
        }

        if state == "arg" {
            if c == ' ' || c == '\t' {
                args = append(args, current)
                current = ""
                state = "start"
            } else {
                current += string(c)
            }
            continue
        }

        if c != ' ' && c != '\t' {
            state = "arg"
            current += string(c)
        }
    }

    if state == "quotes" {
        return []string{}, errors.New(fmt.Sprintf("Unclosed quote in command line: %s", command))
    }

    if current != "" {
        args = append(args, current)
    }

    return args, nil
}

func (node *Node) runCommand(commandStr string) error {
    // To-Do : Add a command to view node's attributes and variables
	commandStr = strings.TrimSuffix(commandStr, "\n")
	arrCommandStr, parseErr := parseCommandLine(commandStr)
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
			httpMsg.MessageType = 1
            key := arrCommandStr[3]
            rawValue := arrCommandStr[4]
            value, marshalErr := json.Marshal(rawValue)
            handle(marshalErr)
            data := map[string][]byte{key:value}
			httpMsg.Data = data
            fmt.Printf("httpMsg %s\n",httpMsg)
            targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
            node.HttpClientReq(httpMsg,targetUrl,"put")
        case "httpGet":
            if len(arrCommandStr)!=4{
                return fmt.Errorf("Usage of httpGet - httpGet <targetIP> <targetPort> <key to query>")
            }
            httpMsg := &Message{}
            httpMsg.SenderIP = node.IP
            httpMsg.SenderPort = node.Port
            httpMsg.MessageType = 0
            key := arrCommandStr[3]
            httpMsg.Query = key
            fmt.Printf("httpMsg %s\n",httpMsg)
            targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
            node.HttpClientReq(httpMsg,targetUrl,"get")
        case "httpGetNode":
            if len(arrCommandStr)!=4{
                return fmt.Errorf("Usage of httpGetNode - httpGetNode <targetIP> <targetPort> <key to query>")
            }
            httpMsg := &Message{}
            httpMsg.SenderIP = node.IP
            httpMsg.SenderPort = node.Port
            httpMsg.MessageType = 0
            key := arrCommandStr[3]
            httpMsg.Query = key
            fmt.Printf("httpMsg %s\n",httpMsg)
            targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
            node.HttpClientReq(httpMsg,targetUrl,"get-node")
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
	const NUMBER_OF_VNODES = 3;
	const MAX_KEY = 100;
    const REPLICATION_FACTOR = 3;

    currentIP, err := externalIP()
    fmt.Printf("Setting Node's IP to be %s\n",currentIP)
    handle(err)
    port := os.Args[1]
    DBPath := os.Args[2]
    NodeNumID,strconverr := strconv.Atoi(os.Args[3])

    if strconverr != nil{
        fmt.Errorf("Failed to convert NodeNumID to int. Please enter an integer")
    }

    ring := ConHash.NewRing(MAX_KEY,REPLICATION_FACTOR)
	conNode := ConHash.NewNode(NodeNumID, NUMBER_OF_VNODES,DBPath,currentIP,port,ring)
    nodeResponseChannel := make(chan interface{})
    nodeTimeoutChannel := make(chan interface{})
    node := Node{conNode,nodeResponseChannel,nodeTimeoutChannel}
	//should with assign the ring to node.ring only when we register with ring?
	//node.RegisterWithRing(node.Ring)
    //For demo purposes, gonna hard code a ring
    const MAXID = 20
    const REPLICATIONFACTOR = 1
    NodeDataArray := make([]ConHash.NodeData,MAXID,MAXID)

    NodeDataArray[1] = ConHash.NodeData{"A0","A",1,"127.0.0.1","8080"}
    NodeDataArray[3]= ConHash.NodeData{"A1","A",3,"127.0.0.1","8080"}
    NodeDataArray[5] = ConHash.NodeData{"A2","A",5,"127.0.0.1","8080"}
    NodeDataArray[7] = ConHash.NodeData{"A3","A",7,"127.0.0.1","8080"}
    NodeDataArray[9] = ConHash.NodeData{"A4","A",9,"127.0.0.1","8080"}
    NodeDataArray[11] = ConHash.NodeData{"B0","B",11,"127.0.0.1","8081"}
    NodeDataArray[13] = ConHash.NodeData{"B1","B",13,"127.0.0.1","8081"}
    NodeDataArray[15] = ConHash.NodeData{"B2","B",15,"127.0.0.1","8081"}
    NodeDataArray[17] = ConHash.NodeData{"B3","B",17,"127.0.0.1","8081"}
    NodeDataArray[19] = ConHash.NodeData{"B4","B",19,"127.0.0.1","8081"}
    


    NodePrefList := map[int][]ConHash.NodeData{
        1:[]ConHash.NodeData{ConHash.NodeData{"B0","B",11,"127.0.0.1","8081"}},
        3:[]ConHash.NodeData{ConHash.NodeData{"B0","B",11,"127.0.0.1","8081"}},
        5:[]ConHash.NodeData{ConHash.NodeData{"B0","B",11,"127.0.0.1","8081"}},
        7:[]ConHash.NodeData{ConHash.NodeData{"B0","B",11,"127.0.0.1","8081"}},
        9:[]ConHash.NodeData{ConHash.NodeData{"B0","B",11,"127.0.0.1","8081"}},
        11:[]ConHash.NodeData{ConHash.NodeData{"A0","A",1,"127.0.0.1","8080"}},
        13:[]ConHash.NodeData{ConHash.NodeData{"A0","A",1,"127.0.0.1","8080"}},
        15:[]ConHash.NodeData{ConHash.NodeData{"A0","A",1,"127.0.0.1","8080"}},
        17:[]ConHash.NodeData{ConHash.NodeData{"A0","A",1,"127.0.0.1","8080"}},
        19:[]ConHash.NodeData{ConHash.NodeData{"A0","A",1,"127.0.0.1","8080"}},
    }
    demoRing := &ConHash.Ring{
        MaxID: MAXID,
        RingNodeDataArray:NodeDataArray,
        NodePrefList:NodePrefList,
        ReplicationFactor: REPLICATIONFACTOR,
    }
    fmt.Printf("Reloading Ring from memory: Ring is %v\n",demoRing)

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

    //Start Stetho - supposed to start in another computer
	s:= Stetho.NewStetho("5000", 1, 5)
	go s.Start()
	time.Sleep(time.Duration(3 * time.Second))
    //Ring Server part
    ringServer := NewRing(*ring, "http://10.12.7.122:5000")
    fmt.Println("ring server register with stetho")
    ringServer.RegisterWithStetho("set-ring")

	for i,nodeData := range node.Ring.RingNodeDataArray{
		if i %2 == 1 && i < 10 {
			ringServer.RegisterNodeWithStetho(nodeData.Port, "add-node")
			time.Sleep(2 * time.Second)
		}

	}

	select {}


	//Start of CLI interactivity
	//reader := bufio.NewReader(os.Stdin)
    //fmt.Printf("Node@%s:%s$ ",node.IP,node.Port)
	//for {
    //    fmt.Printf("Node@%s:%s$ ",node.IP,node.Port)
	//	cmdString, err := reader.ReadString('\n')
	//	if err != nil {
	//		fmt.Fprintln(os.Stderr, err)
	//	}
	//	err = node.runCommand(cmdString)
	//	if err != nil {
	//		fmt.Fprintln(os.Stderr, err)
	//	}
	//}
}
=======
//import (
//	"encoding/json"
//	"fmt"
//	badger "github.com/dgraph-io/badger"
//	"log"
//	"net/http"
//	"os"
//	"os/exec"
//	"strings"
//	"50.041-DistSysProject-BunshinDB/pkg/ConHash"
//	//"bufio"
//	"bytes"
//	"errors"
//	glog "github.com/golang/glog"
//	"net"
//	"strconv"
//)
//
//
//type Message struct{
//    SenderIP string
//    SenderPort string
//    MessageType int
//    Data map[string][]byte //Key-Value pair
//    Query string //Just a key string for receiver to query
//    ResponseCode string //200,404 etc.
//    Timestamp []int //Vector Clock
//}
//
//type Node struct{
//    ConHash.Node
//    ResponseChannel chan interface{}
//    TimeoutChannel chan interface{}
//}
//
//type Ring struct{
//    ConHash.Ring
//    stethoUrl string
//}
//
//
//func handle(err interface{}){
//	if err != nil{
//		log.Fatal(err)
//	}
//}
//
//func (node *Node) Start(){
//    //Open the Badger database located in the node's DB Path
//    // It will be created if it doesn't exist
//    db ,err := badger.Open(badger.DefaultOptions(node.DBPath))
//    handle(err)
//    defer db.Close()
//    node.NodeDB = db
//
//    node.HttpServerStart()
//}
//
//func contains(s []int, e int) bool {
//    for _, a := range s {
//        if a == e {
//            return true
//        }
//    }
//    return false
//}
//
//
//func (node *Node) HttpServerStart(){
//
//	http.HandleFunc("/get", node.GetHandler)
//	http.HandleFunc("/put", node.PutHandler)
//	http.HandleFunc("/new-ring", node.NewRingHandler)
//	http.HandleFunc("/get-node", node.GetNodeHandler)
//	http.HandleFunc("/hb", node.HeartbeatHandler)
//	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s",node.Port), nil))
//}
//
//func (node *Node) GetHandler(w http.ResponseWriter, r *http.Request) {
//    var msg *Message
//    fmt.Println("Get Handler activated")
//
//    w.Header().Set("Content-Type", "application/json")
//    if r.Body == nil {
//        http.Error(w, "Please send a request body", 400)
//        return
//    }
//
//    err := json.NewDecoder(r.Body).Decode(&msg)
//    if err != nil {
//        http.Error(w, err.Error(), 400)
//        return
//    }
//    fmt.Println("Get Handler - Allocating Key")
//    fmt.Println(msg)
//    query := msg.Query
//    ring := node.Ring
//    dstNodeHash, dstNodeUrl , AllocErr := ring.AllocateKey(query)
//    if AllocErr != nil{
//        fmt.Println("Failed to allocate node to key [%s]",query)
//    }
//
//    if contains(node.NodeRingPositions,dstNodeHash){ //If this node is responsible
//        fmt.Println("Get Handler - Retrieving Key Value pair and sending it back to client")
//        var responseStatus string
//        queryResponse, err := node.QueryDB(query)
//        if err != nil{
//            responseStatus = "404"
//        } else {
//            responseStatus = "200"
//        }
//        responseMessage := &Message{
//            SenderIP:node.IP,SenderPort:node.Port,Data:queryResponse,
//            ResponseCode:responseStatus,Timestamp:[]int{},
//        }
//        json.NewEncoder(w).Encode(responseMessage)
//    } else{
//        fmt.Println("Get Handler - Relaying Key to the Coordinator Node")
//        //Need to relay get request to appropriate node
//		//dstNodeData := ring.RingNodeDataArray[dstNodeHash]
//        //dstNodeIPPort := fmt.Sprintf("%s:%s",dstNodeData.IP,dstNodeData.Port)
//        node.HttpClientReq(msg,dstNodeUrl,"get")
//        fmt.Println("Get Handler - Returning relayed message to client")
//        responseMessage := <-node.ResponseChannel
//        fmt.Println("Received Relayed Msg from Coordinator Node")
//        json.NewEncoder(w).Encode(responseMessage)
//    }
//}
//
//
//func (node *Node) PutHandler(w http.ResponseWriter, r *http.Request) {
//    fmt.Println("Put Handler activated")
//    var msg *Message
//
//    w.Header().Set("Content-Type", "application/json")
//    if r.Body == nil {
//        http.Error(w, "Please send a request body", 400)
//        return
//    }
//
//    err := json.NewDecoder(r.Body).Decode(&msg)
//    if err != nil {
//        http.Error(w, err.Error(), 400)
//        return
//    }
//    msgData := msg.Data
//    ring := node.Ring
//    fmt.Println("Put Handler - Allocating Key")
//    for key, _ := range msgData{
//        dstNodeHash, dstNodeUrl , AllocErr := ring.AllocateKey(key) //Get the destination node of this key
//        if AllocErr != nil{
//            fmt.Println("Failed to allocate node to key [%s]",key)
//        }
//        if contains(node.NodeRingPositions,dstNodeHash){ //If this node is responsible
//            fmt.Println("Put Handler - Updating Database with Key Value pair")
//            var responseStatus string
//            err := node.UpdateDB(msgData)
//            if err != nil{
//                responseStatus = "404"
//            } else {
//                responseStatus = "200"
//            }
//            responseMessage := &Message{
//                SenderIP:node.IP,SenderPort:node.Port,Data:msgData,
//                ResponseCode:responseStatus,Timestamp:[]int{},
//            }
//            json.NewEncoder(w).Encode(responseMessage)
//        } else{
//            //Need to relay put request to appropriate node
//            //dstNodeData := ring.RingNodeDataArray[dstNodeHash]
//            //dstNodeIPPort := dstNodeUrl
//
//            node.HttpClientReq(msg,dstNodeUrl,"put")
//            relayResponse := <-node.ResponseChannel
//            relayResponseMsg := relayResponse.(Message)
//            json.NewEncoder(w).Encode(relayResponseMsg)
//        }
//    }
//}
//
//func (node *Node) NewRingHandler(w http.ResponseWriter, r *http.Request) {
//    //To-Do update ring
//    //Need a onUpdateRing function in conHash.go
//}
//
//func (node *Node) GetNodeHandler(w http.ResponseWriter, r *http.Request) {
//    var msg *Message
//
//    w.Header().Set("Content-Type", "application/json")
//    if r.Body == nil {
//        http.Error(w, "Please send a request body", 400)
//        return
//    }
//
//    err := json.NewDecoder(r.Body).Decode(&msg)
//    if err != nil {
//        http.Error(w, err.Error(), 400)
//        return
//    }
//
//    ring := node.Ring
//    query := msg.Query //Get key
//    dstNodeHash, dstNodeUrl, AllocErr := ring.AllocateKey(query)
//    if AllocErr != nil{
//        fmt.Println("Failed to allocate node to key [%s]",query)
//    }
//    responseData := make(map[string][]byte)
//    responseData["key"]=[]byte(query)
//    responseData["nodeId"]=[]byte(ring.RingNodeDataArray[dstNodeHash].ID)
//    responseData["nodeUrl"]=[]byte(dstNodeUrl)
//    responseMessage := &Message{
//        SenderIP:node.IP,SenderPort:node.Port,Data:responseData,
//        ResponseCode:"200",Timestamp:[]int{},
//    }
//    fmt.Println(responseMessage)
//    json.NewEncoder(w).Encode(responseMessage)
//}
//
//func (node *Node) HeartbeatHandler(w http.ResponseWriter, r *http.Request) {
//    w.WriteHeader(http.StatusOK) //Set response code to 200
//    fmt.Fprintf(w,"") //Just send a blank reply at least the server knows you're reachable
//}
//
//func (node *Node) handleMessage(m *Message) *Message{
//    switch m.MessageType{
//    case 0:
//        //If Message type is GET
//        var responseStatus string
//        query := m.Query
//        queryResponse, err := node.QueryDB(query)
//        if err != nil{
//            responseStatus = "404"
//        } else {
//            responseStatus = "200"
//        }
//        responseMessage := &Message{
//            SenderIP:node.IP,SenderPort:node.Port,Data:queryResponse,
//            ResponseCode:responseStatus,Timestamp:[]int{},
//        }
//        return responseMessage
//    case 1:
//        // If Message is to PUT
//        data := m.Data
//        node.UpdateDB(data)
//        responseMessage := &Message{
//            SenderIP:node.IP,SenderPort:node.Port,
//            ResponseCode:"200",Timestamp:[]int{},
//        }
//        return responseMessage
//    }
//    return nil
//}
//
//func (node *Node) HttpClientReq(msg *Message,targetUrl string,endpoint string){
//	client := &http.Client{
//	}
//    fmt.Println("HTTP Client Req function called")
//    url := fmt.Sprintf("http://%s/%s",targetUrl,endpoint)
//
//    jsonBuffer, err := json.Marshal(msg)
//    handle(err)
//
//	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBuffer))
//    req.Header.Set("Content-Type", "application/json")
//
//    res, err := client.Do(req)
//    defer res.Body.Close()
//    fmt.Println("HTTP Client Req - Got a response")
//
//    // always close the response-body, even if content is not required
//
//    if err != nil {
//         fmt.Println("Unable to reach the server.")
//    } else {
//        var resMsg Message
//		json.NewDecoder(res.Body).Decode(&resMsg)
//        fmt.Printf("Response Message is \n%v\n",resMsg)
//        go func(){
//            node.ResponseChannel <- resMsg
//        }()
//        //select{
//        //case node.ResponseChannel <- resMsg:
//        //    fmt.Println("Hello hello my name's dibo")
//        //        // Do nothing
//        //case <- time.After(100 * time.Millisecond):
//        //    fmt.Println("Hello hello my name's nobo")
//        //    //Do nothing
//        //}
//    }
//}
//
//
//
//func (node *Node) UpdateDB(update map[string][]byte) error{
//    db := node.NodeDB
//    txn := db.NewTransaction(true)
//    for k,v := range update{
//      if err := txn.Set([]byte(k),[]byte(v)); err == badger.ErrTxnTooBig {
//        _ = txn.Commit()
//        txn = db.NewTransaction(true)
//        _ = txn.Set([]byte(k),[]byte(v))
//      }
//    }
//    err := txn.Commit()
//    return err
//}
//
////print all key, value pairs
//func (node *Node) ViewDB(){
//    db := node.NodeDB
//	err := db.View(func(txn *badger.Txn) error {
//	  opts := badger.DefaultIteratorOptions
//	  opts.PrefetchSize = 10
//	  it := txn.NewIterator(opts)
//	  defer it.Close()
//	  for it.Rewind(); it.Valid(); it.Next() {
//	    item := it.Item()
//	    k := item.Key()
//	    err := item.Value(func(v []byte) error {
//	      fmt.Printf("key=%s, value=%s\n", k, v)
//	      return nil
//	    })
//	    if err != nil {
//	      return err
//	    }
//	  }
//	  return nil
//	})
//    handle(err)
//}
//
//func (node *Node) QueryDB(queryKey string) (map[string][]byte,error){
//	var outputVal []byte
//    var valCopy []byte
//    db := node.NodeDB
//	err := db.View(func(txn *badger.Txn) error {
//	item, err := txn.Get([]byte(queryKey))
//    if err!=nil{
//        glog.Error(err)
//	    return err
//    }
//
//	//var valCopy []byte
//	err = item.Value(func(val []byte) error {
//	// This func with val would only be called if item.Value encounters no error.
//
//	// Copying or parsing val is valid.
//	valCopy = append([]byte{}, val...)
//
//	return nil
//	})
//
//    if err!=nil{
//        glog.Error(err)
//	    return err
//    }
//
//	// You must copy it to use it outside item.Value(...).
//	fmt.Printf("The answer is: %s\n", valCopy)
//
//	return nil
//	})
//
//    outputVal = valCopy
//    output := make(map[string][]byte)
//    output[queryKey]=outputVal
//	return output, err
//}
//
//func (node *Node) DeleteKey(Key string) error{
//    db := node.NodeDB
//	err := db.Update(func(txn *badger.Txn) error {
//	err := txn.Delete([]byte(Key))
//	if err!=nil{
//        return err
//    }
//
//	return nil
//	})
//    return err
//}
//
//func externalIP() (string, error) {
//	ifaces, err := net.Interfaces()
//	if err != nil {
//		return "", err
//	}
//	for _, iface := range ifaces {
//		if iface.Flags&net.FlagUp == 0 {
//			continue // interface down
//		}
//		if iface.Flags&net.FlagLoopback != 0 {
//			continue // loopback interface
//		}
//		addrs, err := iface.Addrs()
//		if err != nil {
//			return "", err
//		}
//		for _, addr := range addrs {
//			var ip net.IP
//			switch v := addr.(type) {
//			case *net.IPNet:
//				ip = v.IP
//			case *net.IPAddr:
//				ip = v.IP
//			}
//			if ip == nil || ip.IsLoopback() {
//				continue
//			}
//			ip = ip.To4()
//			if ip == nil {
//				continue // not an ipv4 address
//			}
//			return ip.String(), nil
//		}
//	}
//	return "", errors.New("are you connected to the network?")
//}
//
//func parseCommandLine(command string) ([]string, error) {
//	//Finite state machine to handle arguments with white spaces enclosed within quotes
//	//Handles escaped stuff too
//    var args []string
//    state := "start"
//    current := ""
//    quote := "\""
//    escapeNext := true
//    for i := 0; i < len(command); i++ {
//        c := command[i]
//
//        if state == "quotes" {
//            if string(c) != quote {
//                current += string(c)
//            } else {
//                args = append(args, current)
//                current = ""
//                state = "start"
//            }
//            continue
//        }
//
//        if (escapeNext) {
//            current += string(c)
//            escapeNext = false
//            continue
//        }
//
//        if (c == '\\') {
//            escapeNext = true
//            continue
//        }
//
//        if c == '"' || c == '\'' {
//            state = "quotes"
//            quote = string(c)
//            continue
//        }
//
//        if state == "arg" {
//            if c == ' ' || c == '\t' {
//                args = append(args, current)
//                current = ""
//                state = "start"
//            } else {
//                current += string(c)
//            }
//            continue
//        }
//
//        if c != ' ' && c != '\t' {
//            state = "arg"
//            current += string(c)
//        }
//    }
//
//    if state == "quotes" {
//        return []string{}, errors.New(fmt.Sprintf("Unclosed quote in command line: %s", command))
//    }
//
//    if current != "" {
//        args = append(args, current)
//    }
//
//    return args, nil
//}
//
//func (node *Node) runCommand(commandStr string) error {
//    // To-Do : Add a command to view node's attributes and variables
//	commandStr = strings.TrimSuffix(commandStr, "\n")
//	arrCommandStr, parseErr := parseCommandLine(commandStr)
//	handle(parseErr)
//
//    //Subcommands
//    if len(arrCommandStr)>=1{
//		switch arrCommandStr[0] {
//		case "exit":
//			os.Exit(0)
//			// add another case here for custom commands.
//        case "help":
//            fmt.Printf(
//`
//Here are the list of commands:
//
//help: Shows lists of commands
//
//exit: quits program
//
//query: Usage - query <key>
//query searches the database for a key and returns the value
//
//update: Usage - update <key> <value>
//update takes user inputted key value pair and updates the database
//
//view: Views database
//
//delete: Usage - delete <key>
//delete deletes an existing key and its respective value from database
//
//allocateKey: Usage - allocateKey <Key>
//allocateKey searches the node's consistent hash ring and find out which node is
//the coordinator node which is responsible for the read/write for the given key
//
//httpPut: Usage - httpPut <targetIP> <targetPort> <key> <value>
//httpPut sends user inputted data to another server and updates their database
//
//httpGet: Usage - httpGet <targetIP> <targetPort> <key>
//httpGet sends a key to another server, the receiving server will query its database
//for the key-value pair and it in the response
//
//httpGetNode: Usage - httpGetNode <targetIP> <targetPort> <key>
//httpGetNode sends a key to another server, the receiving server will refer to the consistent
//hash ring and find out which coordinator node is responsible for the read/writes of this key.
//Then, the server will return a message to the client containing the Node ID and the Node's IP
//address and port
//
//`)
//        case "query":
//            if len(arrCommandStr)!=2{
//                return fmt.Errorf("Usage of query - query <Key>")
//            }
//            key := arrCommandStr[1]
//            fmt.Printf("Querying db...\n")
//            results,err := node.QueryDB(key)
//            if err!=nil{
//                fmt.Printf("Key <%s> not found in datbase\n",key)
//            } else{
//            fmt.Printf("Query results are %s\n",results)
//            }
//        case "update":
//           if len(arrCommandStr)!=3{
//               return fmt.Errorf("Usage of update - update <key> <value>")
//           }
//           key := arrCommandStr[1]
//           rawValue := arrCommandStr[2]
//           value, marshalErr := json.Marshal(rawValue)
//           handle(marshalErr)
//           update := map[string][]byte{key:value}
//           node.UpdateDB(update)
//        case "view":
//            if len(arrCommandStr)!=1{
//                return fmt.Errorf("Extra arguments, usage of view - view")
//            }
//            node.ViewDB()
//        case "delete":
//            if len(arrCommandStr)!=2{
//                return fmt.Errorf("Usage of delete - delete <Key>")
//            }
//            key := arrCommandStr[1]
//            err := node.DeleteKey(key)
//            if err!=nil{
//                fmt.Printf("Key <%s> not in database, can't delete\n",key)
//            } else {
//                fmt.Printf("Key <%s> has been deleted from database\n",key)
//            }
//        case "allocateKey":
//            if len(arrCommandStr)!=2{
//                return fmt.Errorf("Usage of allocateKey - allocateKey <Key>")
//            }
//            key := arrCommandStr[1]
//            dstNodeHash, dstNodeUrl, AllocErr := node.Ring.AllocateKey(key)
//            if AllocErr!=nil{
//                fmt.Printf("Failed to allocate key to a Coordinator Node\n",key)
//            } else {
//                fmt.Printf("Coordinator for key <%s> is Node %s and its url is %s\n",key,node.Ring.RingNodeDataArray[dstNodeHash].ID ,dstNodeUrl)
//            }
//		case "httpPut":
//			//Do nothing
//			if len(arrCommandStr)!=5{
//				return fmt.Errorf("Usage of httpPut - httpPut <targetIP> <targetPort> <key> <value")
//			}
//			httpMsg := &Message{}
//			httpMsg.SenderIP = node.IP
//			httpMsg.SenderPort = node.Port
//			httpMsg.MessageType = 1
//            key := arrCommandStr[3]
//            rawValue := arrCommandStr[4]
//            value, marshalErr := json.Marshal(rawValue)
//            handle(marshalErr)
//            data := map[string][]byte{key:value}
//			httpMsg.Data = data
//            fmt.Printf("httpMsg %s\n",httpMsg)
//            targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
//            node.HttpClientReq(httpMsg,targetUrl,"put")
//        case "httpGet":
//            if len(arrCommandStr)!=4{
//                return fmt.Errorf("Usage of httpGet - httpGet <targetIP> <targetPort> <key to query>")
//            }
//            httpMsg := &Message{}
//            httpMsg.SenderIP = node.IP
//            httpMsg.SenderPort = node.Port
//            httpMsg.MessageType = 0
//            key := arrCommandStr[3]
//            httpMsg.Query = key
//            fmt.Printf("httpMsg %s\n",httpMsg)
//            targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
//            node.HttpClientReq(httpMsg,targetUrl,"get")
//        case "httpGetNode":
//            if len(arrCommandStr)!=4{
//                return fmt.Errorf("Usage of httpGetNode - httpGetNode <targetIP> <targetPort> <key to query>")
//            }
//            httpMsg := &Message{}
//            httpMsg.SenderIP = node.IP
//            httpMsg.SenderPort = node.Port
//            httpMsg.MessageType = 0
//            key := arrCommandStr[3]
//            httpMsg.Query = key
//            fmt.Printf("httpMsg %s\n",httpMsg)
//            targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
//            node.HttpClientReq(httpMsg,targetUrl,"get-node")
//        default:
//		cmd := exec.Command(arrCommandStr[0], arrCommandStr[1:]...)
//		cmd.Stderr = os.Stderr
//		cmd.Stdout = os.Stdout
//		return cmd.Run()
//    }
//}
//    return nil
//}
//
//
//func main(){
//
//    if len(os.Args) != 4{
//        fmt.Printf("Usage of program is: %s <PORT> <DBPath> <NodeNumID>\n", os.Args[0])
//        os.Exit(0)
//    }
//	//Set constants here
//	const NUMBER_OF_VNODES = 3;
//	const MAX_KEY = 100;
//    const REPLICATION_FACTOR = 3;
//
//    currentIP, err := externalIP()
//    fmt.Printf("Setting Node's IP to be %s\n",currentIP)
//    handle(err)
//    port := os.Args[1]
//    DBPath := os.Args[2]
//    NodeNumID,strconverr := strconv.Atoi(os.Args[3])
//
//    if strconverr != nil{
//        fmt.Errorf("Failed to convert NodeNumID to int. Please enter an integer")
//    }
//
//    ring := ConHash.NewRing(MAX_KEY,REPLICATION_FACTOR, 1)
//	conNode := ConHash.NewNode(NodeNumID, NUMBER_OF_VNODES,DBPath,currentIP,port,ring)
//    nodeResponseChannel := make(chan interface{})
//    nodeTimeoutChannel := make(chan interface{})
//    node := Node{conNode,nodeResponseChannel,nodeTimeoutChannel}
//	//should with assign the ring to node.ring only when we register with ring?
//	//node.RegisterWithRing(node.Ring)
//    //For demo purposes, gonna hard code a ring
//    const MAXID = 20
//    const REPLICATIONFACTOR = 1
//    NodeDataArray := make([]ConHash.NodeData,MAXID,MAXID)
//
//    NodeDataArray[1] = ConHash.NodeData{"A0","A",1,"127.0.0.1","8080"}
//    NodeDataArray[3]= ConHash.NodeData{"A1","A",3,"127.0.0.1","8080"}
//    NodeDataArray[5] = ConHash.NodeData{"A2","A",5,"127.0.0.1","8080"}
//    NodeDataArray[7] = ConHash.NodeData{"A3","A",7,"127.0.0.1","8080"}
//    NodeDataArray[9] = ConHash.NodeData{"A4","A",9,"127.0.0.1","8080"}
//    NodeDataArray[11] = ConHash.NodeData{"B0","B",11,"127.0.0.1","8081"}
//    NodeDataArray[13] = ConHash.NodeData{"B1","B",13,"127.0.0.1","8081"}
//    NodeDataArray[15] = ConHash.NodeData{"B2","B",15,"127.0.0.1","8081"}
//    NodeDataArray[17] = ConHash.NodeData{"B3","B",17,"127.0.0.1","8081"}
//    NodeDataArray[19] = ConHash.NodeData{"B4","B",19,"127.0.0.1","8081"}
//
//
//
//
//    NodePrefList := map[int][]ConHash.NodeData{
//        1:[]ConHash.NodeData{ConHash.NodeData{"B0","B",11,"127.0.0.1","8081"}},
//        3:[]ConHash.NodeData{ConHash.NodeData{"B0","B",11,"127.0.0.1","8081"}},
//        5:[]ConHash.NodeData{ConHash.NodeData{"B0","B",11,"127.0.0.1","8081"}},
//        7:[]ConHash.NodeData{ConHash.NodeData{"B0","B",11,"127.0.0.1","8081"}},
//        9:[]ConHash.NodeData{ConHash.NodeData{"B0","B",11,"127.0.0.1","8081"}},
//        11:[]ConHash.NodeData{ConHash.NodeData{"A0","A",1,"127.0.0.1","8080"}},
//        13:[]ConHash.NodeData{ConHash.NodeData{"A0","A",1,"127.0.0.1","8080"}},
//        15:[]ConHash.NodeData{ConHash.NodeData{"A0","A",1,"127.0.0.1","8080"}},
//        17:[]ConHash.NodeData{ConHash.NodeData{"A0","A",1,"127.0.0.1","8080"}},
//        19:[]ConHash.NodeData{ConHash.NodeData{"A0","A",1,"127.0.0.1","8080"}},
//    }
//    demoRing := &ConHash.Ring{
//        MaxID: MAXID,
//        RingNodeDataArray:NodeDataArray,
//        NodePrefList:NodePrefList,
//        ReplicationFactor: REPLICATIONFACTOR,
//    }
//    fmt.Printf("Reloading Ring from memory: Ring is %v\n",demoRing)
//
//    node.Ring = demoRing
//    for _,nodeData := range node.Ring.RingNodeDataArray{
//        if nodeData.CName == node.CName{
//            node.NodeRingPositions = append(node.NodeRingPositions,nodeData.Hash)
//        }
//    }
//    fmt.Println(node.NodeRingPositions)
//
//    nodeQuery := "A2"
//	nodeIP, err := demoRing.GetNode(nodeQuery)
//	if err == nil {
//        fmt.Printf("Node %s found at : %s \n",nodeQuery,nodeIP)
//    } else{
//        fmt.Printf("Node %s not found\n",nodeQuery)
//    }
//
//    searchKey := "testing"
//    nodeHash, addr, err := demoRing.AllocateKey(searchKey)
//    if err == nil {
//		fmt.Printf("Key [%s] found at node %s with ip [%s] \n",searchKey, demoRing.RingNodeDataArray[nodeHash].ID,addr)
//	} else {
//		fmt.Printf("Node for key [%s] not found \n", searchKey )
//	}
//
//
//    go node.Start()
//
//    //Start Stetho - supposed to start in another computer
//	//s:= Stetho.NewStetho("50	00", 1, 5)
//	//go s.Start()
//	//time.Sleep(time.Duration(3 * time.Second))
//    ////Ring Server part
//    //ringServer := NewRing(*ring, "http://10.12.7.122:5000", 1)
//    //fmt.Println("ring server register with stetho")
//    //ringServer.RegisterWithStetho("set-ring")
//	//
//	//for i,nodeData := range node.Ring.RingNodeDataArray{
//	//	if i %2 == 1 && i < 10 {
//	//		ringServer.RegisterNodeWithStetho(nodeData.Port, "add-node")
//	//		time.Sleep(2 * time.Second)
//	//	}
//	//
//	//}
//
//	select {}
//
//
//	//Start of CLI interactivity
//	//reader := bufio.NewReader(os.Stdin)
//    //fmt.Printf("Node@%s:%s$ ",node.IP,node.Port)
//	//for {
//    //    fmt.Printf("Node@%s:%s$ ",node.IP,node.Port)
//	//	cmdString, err := reader.ReadString('\n')
//	//	if err != nil {
//	//		fmt.Fprintln(os.Stderr, err)
//	//	}
//	//	err = node.runCommand(cmdString)
//	//	if err != nil {
//	//		fmt.Fprintln(os.Stderr, err)
//	//	}
//	//}
//}
>>>>>>> master

