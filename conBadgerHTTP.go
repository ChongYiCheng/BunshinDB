package main

import (
    "encoding/json"
    "log"
    "fmt"
    badger "github.com/dgraph-io/badger"
    "net/http"
    "os"
    "os/exec"
    "strings"
    "bufio"
    "bytes"
    "net"
    "errors"
    glog "github.com/golang/glog"
    "crypto/md5"
)

type Node struct{
    id string
	cName string
    numTokens int
	//quitChannel chan struct{}
    //nodeChannel chan interface{}
    DBPath string // e.g /tmp/badger
    nodeDB *badger.DB
    ip string //a.b.c.d:port
    port string
    //allNodes map[int]string
    //localClock []int
    ring *Ring
	nodeRingPositions []int
	//nodeDataArray []nodeData
}

type Ring struct{
    maxID int // 0 to maxID inclusive
    nodeArray []nodeData
}

type nodeData struct{
    //Node Data contains ip and port and hash This helps in determining which node is responsible for
    //Which request(read/write) and contains relevant info (ip:port) to
    //direct data to that node
    id string
	cName string
    hash int
    ip string
    port string
}

type Message struct{
    SenderIP string
    SenderPort string
    MessageType int
    Data map[string][]byte //Key-Value pair
    Query string //Just a key string for receiver to query
    ResponseCode string //200,404 etc.
    Timestamp []int //Vector Clock
}

func toChar(i int) rune {
	return rune('A' - 1 + i)
}


func newNodeData(id string, cName string, hash int,ip string, port string) nodeData{
    return nodeData{id, cName, hash, ip, port}
}

func newNode(numID int, numTokens int, DBPath string, ip string, port string, ring *Ring) *Node{
    return &Node{
        id:string(toChar(numID)) + "0",
        cName:string(toChar(numID)),
        numTokens: numTokens,
        DBPath: DBPath,
        ip: ip,
        port: port,
        ring: ring,
    }
}

func newRing(maxID int) *Ring{
    nodeDataArray := make([]nodeData, maxID, maxID)
    fmt.Println(len(nodeDataArray))
    fmt.Println(nodeDataArray[1].id)
    return &Ring{maxID, nodeDataArray}
}

//node will create numTokens worth of virtual nodes
func (n *Node) registerWithRing(r *Ring) {
    localRing := n.ring
	nodeAddresses := []int {}
    tempNodeDataArray := make([]nodeData, len(localRing.nodeArray),len(localRing.nodeArray))
    copy(tempNodeDataArray,localRing.nodeArray)
	//TODO: Can we do deduplication on the node side?
	for i := 0; i < n.numTokens +1; i ++ {
		id := fmt.Sprintf("%s%d", n.cName, i)
		hash := hashMD5(id, 0, r.maxID)
		nodeAddresses = append(nodeAddresses, hash)
		tempNodeDataArray = append(tempNodeDataArray, newNodeData(id, n.cName, hash, n.ip, n.port))
	}
	fmt.Printf("Node %s registering %s \n", n.id, toString(tempNodeDataArray))
	r.nodeArray = r.registerNodes(tempNodeDataArray)
	fmt.Printf("Ring registered for %s: %s  \n", n.id, toString(tempNodeDataArray))
}

func (r *Ring) registerNodes(nodeDataArray []nodeData) []nodeData{
	ret := []nodeData{}
	for _, nd := range nodeDataArray {
		for {
			//if occupied, we do linear probing
			if r.nodeArray[nd.hash].id != "" {
				nd.hash = (nd.hash + 1) % len(nodeDataArray)
			} else {
				r.nodeArray[nd.hash] = nd
				ret = append(ret, nd)
				break
			}

		}
	}
	return ret
}

//Easy toString method
func toString(nodeDataArray []nodeData) []string{
	ret := []string {}
	for _, nd := range nodeDataArray {
		ret = append(ret, fmt.Sprintf("(%s, %d) ", nd.id, nd.hash))
	}
	return ret
}

//string must end with an integer
func (r *Ring) getNode(id string) (string, error) {
	var NodeNotFound = errors.New("Node not found")
	hash := hashMD5(id, 0, r.maxID)

	//Impose an upper bound for probe times
	for i:= 0; i < len(r.nodeArray); i ++{
		fmt.Println(r.nodeArray[hash].id)
		if r.nodeArray[hash].id == id {
            ip_port := fmt.Sprintf("%s:%s",r.nodeArray[hash].ip,r.nodeArray[hash].port)
			//return r.nodeArray[hash].physicalNode, nil
            return ip_port, nil
		}
		hash = (hash + 1) % len(r.nodeArray)
	}

	return id, NodeNotFound
}

func hashMD5(text string, min int, max int) int {
	byteArray := md5.Sum([] byte(text))
	var output int
	for _, num := range byteArray{
		output += int(num)
	}

	return output % (max - min + 1) + min
}

//Todo function
//func allocateKey(key String, ring *Ring){
//    nodeArray := ring.nodeArray
//    keyHash = hashMD5(key,0,len(nodeArray)-1)
//    firstNodeAddress := ""
//    for i := 0; i < len(nodeArray) ; i++ {
//        if nodeArray[i].id != "" {
//            if keyHash <= i{
//                httpSend //Todo
//                return
//            }
//        }
//        if i == len(nodeArray)-1 and nodeArray[i].id != ""{
//            httpSend(firstNodeAddress) //Todo
//            return
//        }
//    }
//}

func (ring *Ring) genPrefList(){
    //To Do
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
    node.nodeDB = db

    node.httpServerStart()
}

func (node *Node) httpServerStart(){
    var msg Message

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
        fmt.Println(msg)
        response := node.handleMessage(&msg)
        json.NewEncoder(w).Encode(response)
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s",node.port), nil))
}

func (node *Node) handleMessage(m *Message) *Message{
    switch m.MessageType{
    case 0:
        //If Message type is GET
        var responseStatus string
        query := m.Query
        queryResponse, err := node.queryDB(query)
        if err != nil{
            responseStatus = "404"
        } else {
            responseStatus = "200"
        }
        responseMessage := &Message{
            SenderIP:node.ip,SenderPort:node.port,Data:queryResponse,
            ResponseCode:responseStatus,Timestamp:[]int{},
        }
        return responseMessage
    case 1:
        // If Message is to PUT
        data := m.Data
        node.UpdateDB(data)
        responseMessage := &Message{
            SenderIP:node.ip,SenderPort:node.port,
            ResponseCode:"200",Timestamp:[]int{},
        }
        return responseMessage
    }
    return nil
}

func (node *Node) httpClientReq(msg *Message,targetUrl string){
	client := &http.Client{
	}

    url := fmt.Sprintf("http://%s/",targetUrl)
    fmt.Println(msg)

    jsonBuffer, err := json.Marshal(msg)
    handle(err)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBuffer))
    req.Header.Set("Content-Type", "application/json")

    res, err := client.Do(req)
    if err != nil {
         fmt.Println("Unable to reach the server.")
    } else {
        var resMsg Message
		json.NewDecoder(res.Body).Decode(&resMsg)
        fmt.Println(resMsg)
    }
}



func (node *Node) UpdateDB(update map[string][]byte){
    db := node.nodeDB
    txn := db.NewTransaction(true)
    for k,v := range update{
      if err := txn.Set([]byte(k),[]byte(v)); err == badger.ErrTxnTooBig {
        _ = txn.Commit()
        txn = db.NewTransaction(true)
        _ = txn.Set([]byte(k),[]byte(v))
      }
    }
    err := txn.Commit()
    handle(err)
}

func (node *Node) viewDB(){
    db := node.nodeDB
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

func (node *Node) queryDB(queryKey string) (map[string][]byte,error){
	var outputVal []byte
    var valCopy []byte
    db := node.nodeDB
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

func (node *Node) deleteKey(Key string) error{
    db := node.nodeDB
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
help : Shows lists of commands

exit : quits program

query : Usage - query <key>
query searches the database for a key and returns the value

update : Usage - update <key> <value>
update takes user inputted key value pair and updates the database

view : Views database

delete : Usage - delete <key>
delete deletes an existing key and its respective value from database

httpPut : Usage - httpPut <targetIP> <targetPort> <key> <value>
httpPut sends user inputted data to another server and updates their database

httpGet : Usage - httpGet <targetIP> <targetPort> <key>
httpGet sends a key to another server, the receiving server will query its database
for the key-value pair and it in the response

`)
        case "query":
            if len(arrCommandStr)!=2{
                return fmt.Errorf("Usage of query - query <Key>")
            }
            key := arrCommandStr[1]
            fmt.Printf("Querying db...\n")
            results,err := node.queryDB(key)
            if err!=nil{
                fmt.Printf("Key <%s> not found in datbase\n",key)
            } else{
            fmt.Printf("Query results are %s",results)
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
            node.viewDB()
        case "delete":
            if len(arrCommandStr)!=2{
                return fmt.Errorf("Usage of delete - delete <Key>")
            }
            key := arrCommandStr[1]
            err := node.deleteKey(key)
            if err!=nil{
                fmt.Printf("Key <%s> not in database, can't delete",key)
            } else {
                fmt.Printf("Key <%s> has been deleted from database\n",key)
            }
		case "httpPut":
			//Do nothing
			if len(arrCommandStr)!=5{
				return fmt.Errorf("Usage of httpSend - httpSend <targetIP> <targetPort> <key> <value")
			}
			httpMsg := &Message{}
			httpMsg.SenderIP = node.ip
			httpMsg.SenderPort = node.port
			httpMsg.MessageType = 1
            key := arrCommandStr[3]
            rawValue := arrCommandStr[4]
            value, marshalErr := json.Marshal(rawValue)
            handle(marshalErr)
            data := map[string][]byte{key:value}
			httpMsg.Data = data
            fmt.Printf("httpMsg %s\n",httpMsg)
            targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
            node.httpClientReq(httpMsg,targetUrl)
        case "httpGet":
            if len(arrCommandStr)!=4{
                return fmt.Errorf("Usage of httpGet - httpGet <targetIP> <targetPort> <key to query>")
            }
            httpMsg := &Message{}
            httpMsg.SenderIP = node.ip
            httpMsg.SenderPort = node.port
            httpMsg.MessageType = 0
            key := arrCommandStr[3]
            httpMsg.Query = key
            fmt.Printf("httpMsg %s\n",httpMsg)
            targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
            node.httpClientReq(httpMsg,targetUrl)
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

    if len(os.Args) != 3{
        fmt.Printf("Usage of program is: %s , <PORT> <DBPath>\n", os.Args[0])
        os.Exit(0)
    }
	//Set constants here
	const NUMBER_OF_VNODES = 4;
	const MAX_KEY = 20

    currentIP, err := externalIP()
	fmt.Printf("Setting Node's IP to be %s\n",currentIP)
    handle(err)
    port := os.Args[1]
    DBPath := os.Args[2]
	
	r := newRing(MAX_KEY)
	node := newNode(1, NUMBER_OF_VNODES,DBPath,currentIP,port,r)
	node.registerWithRing(node.ring)

    nodeQuery := "A2"
	nodeIP, err := r.getNode("A2")
	if err == nil {
        fmt.Printf("Node %s found at : %s \n",nodeQuery,nodeIP)
    } else{
        fmt.Printf("Node %s not found\n",nodeQuery)
    }
    //node.name = "A0"
    //currentIP, err := externalIP()
    //node.ip = currentIP
	//fmt.Printf("Setting Node's IP to be %s\n",node.ip)
    //handle(err)
    //node.port = os.Args[1]
    //node.DBPath = os.Args[2]
    //node.quitChannel = make(chan struct{})
    //node.nodeChannel = make(chan interface{})
    //node.allNodes = make(map[int]string)
    //node.localClock = []int{0}
    go node.Start()

	//Start of CLI interactivity
	reader := bufio.NewReader(os.Stdin)
    fmt.Printf("Node@%s:%s$ ",node.ip,node.port)
	for {
        fmt.Printf("Node@%s:%s$ ",node.ip,node.port)
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

