package main

import (
	badger "github.com/dgraph-io/badger"
	"net/http"
	"encoding/json"
    "log"
	"fmt"
	"strings"
	"os"
	"os/exec"
	"bytes"
	glog "github.com/golang/glog"
)

type Server struct{
    DBPath string // e.g /tmp/badger
    nodeDB *badger.DB
    ip string 
    port string
    serverList []string
}



func (server *Server) Start(){
    //Open the Badger database located in the node's DB Path
	// It will be created if it doesn't exist
	db ,err := badger.Open(badger.DefaultOptions(server.DBPath))
	handle(err)
	defer db.Close()
	server.nodeDB = db
    server.httpServerStart()
}

func (server *Server) httpServerStart(){
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
		response := server.handleMessage(&msg)
		fmt.Println(response)
        json.NewEncoder(w).Encode(response)
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s",server.port), nil))
}



func (server *Server) ServerBroadcastRead(targetIP string, targetPort string, waitChan chan int) {
	httpMsg := &Message{}
	httpMsg.SenderIP = server.ip
	httpMsg.SenderPort = server.port
	httpMsg.MessageType = "SERVER_READ"
	targetUrl := fmt.Sprintf("%s:%s",targetIP,targetPort)
	server.httpServerReq(httpMsg,targetUrl, waitChan)
}



func (server *Server) ServerBroadcastWrite(data map[string][]byte, targetIP string, targetPort string, waitChan chan int) {
	httpMsg := &Message{}
	httpMsg.SenderIP = server.ip
	httpMsg.SenderPort = server.port
	httpMsg.MessageType = "SERVER_WRITE"
	httpMsg.Data = data
	targetUrl := fmt.Sprintf("%s:%s",targetIP,targetPort)
	server.httpServerReq(httpMsg,targetUrl, waitChan)
}



func (server *Server) httpServerReq(msg *Message,targetUrl string, waitChan chan int){
	client_ := &http.Client{
	}

    url := fmt.Sprintf("http://%s/",targetUrl)
    fmt.Println(msg)

    jsonBuffer, err := json.Marshal(msg)
    handle(err)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBuffer))
    req.Header.Set("Content-Type", "application/json")

    res, err := client_.Do(req)
    if err != nil {
         fmt.Println("Unable to reach the server.")
    } else {
        var resMsg Message
		json.NewDecoder(res.Body).Decode(&resMsg)
		fmt.Println(resMsg)
		waitChan <- 1
    }
}




func (server *Server) handleMessage(m *Message) *Message{
    switch m.MessageType{
	case "SERVER_WRITE":
		//save to db
		data := m.Data
        server.UpdateDB(data)
		responseMessage := &Message{
			SenderIP: server.ip,
			SenderPort:server.port,
			// Data:queryResponse,    //might have error if no data field ????
			ResponseCode:"200",
		}
		fmt.Println("sending response to coordinator")
        return responseMessage

	case "SERVER_READ":
		responseMessage := &Message{
			SenderIP: server.ip,
			SenderPort:server.port,
			// Data:queryResponse,    //might have error if no data field ????
			ResponseCode:"200",
		}
		fmt.Println("sending response to coordinator")
        return responseMessage
    case "GET":
		R:=2
		waitChan := make(chan int, 9)
		for i:=0; i<len(server.serverList);i++ {
			if server.serverList[i] != server.port {
				server.ServerBroadcastRead(server.ip, server.serverList[i], waitChan)
			}
		}
		for {
			if len(waitChan) == R {
				fmt.Println("COORDINATOR HAS RECEIVED R RESPONSES")
				break
			}
		}
        var responseStatus string
        query := m.Query
        queryResponse, err := server.queryDB(query)
        if err != nil{
            responseStatus = "404"
        } else {
            responseStatus = "200"
        }
        responseMessage := &Message{
			SenderIP: server.ip,
			SenderPort:server.port,
			Data:queryResponse,
			ResponseCode:responseStatus,
		}
        return responseMessage
	case "POST":
        data := m.Data
        server.UpdateDB(data)
        responseMessage := &Message{
			SenderIP: server.ip,
			SenderPort: server.port,
            ResponseCode:"200",
		}
		
		W := 2
		waitChan := make(chan int, 9)
		for i:=0; i<len(server.serverList);i++ {
			if server.serverList[i] != server.port {
				server.ServerBroadcastWrite(data, server.ip, server.serverList[i], waitChan)
			}
		}
		for {
			if len(waitChan) == W {
				fmt.Println("COORDINATOR HAS RECEIVED W RESPONSES")
				break
			}
		}

        return responseMessage
    }
    return nil
}



func (server *Server) UpdateDB(update map[string][]byte){
    db := server.nodeDB
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


func (server *Server) viewDB(){
    db := server.nodeDB
	err := db.View(func(txn *badger.Txn) error {
	  opts := badger.DefaultIteratorOptions
	  opts.PrefetchSize = 10
	  it := txn.NewIterator(opts)
	  defer it.Close()
	  for it.Rewind(); it.Valid(); it.Next() {
	    item := it.Item()
	    k := item.Key()
	    err := item.Value(func(v []byte) error {
	      fmt.Printf("key=%s, value=%s\r\n", k, v)
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

func (server *Server) queryDB(queryKey string) (map[string][]byte,error){
	var outputVal []byte
    var valCopy []byte
    db := server.nodeDB
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
        fmt.Printf("The answer is: %s\r\n", valCopy)

        return nil
    })

    outputVal = valCopy
    output := make(map[string][]byte)
    output[queryKey]=outputVal
	return output, err
}



func (server *Server) deleteKey(Key string) error{
    db := server.nodeDB
	err := db.Update(func(txn *badger.Txn) error {
	err := txn.Delete([]byte(Key))
	if err!=nil{
        return err
    }

	return nil
	})
    return err
}


func (server *Server) runCommand(commandStr string) error {
	commandStr = strings.TrimSuffix(commandStr, "\r\n")
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
            fmt.Printf("Querying db...\r\n")
            results,err := server.queryDB(key)
            if err!=nil{
                fmt.Printf("Key <%s> not found in datbase\r\n",key)
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
           server.UpdateDB(update)
        case "view":
            if len(arrCommandStr)!=1{
                return fmt.Errorf("Extra arguments, usage of view - view")
            }
            server.viewDB()
        case "delete":
            if len(arrCommandStr)!=2{
                return fmt.Errorf("Usage of delete - delete <Key>")
            }
            key := arrCommandStr[1]
            err := server.deleteKey(key)
            if err!=nil{
                fmt.Printf("Key <%s> not in database, can't delete",key)
            } else {
                fmt.Printf("Key <%s> has been deleted from database\r\n",key)
			}
			
        default:
		cmd := exec.Command(arrCommandStr[0], arrCommandStr[1:]...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		return cmd.Run()
    }
}
    return nil
}


