package main

import (
    "encoding/json"
	"log"
    "strconv"
    "fmt"
	badger "github.com/dgraph-io/badger"
)

type JSON struct{
    ID int
    Name string
}

type Node struct{
    id int
    name string
    quitChannel chan interface{}
    nodeChannel chan interface{}
    nodeDBPath string
    allNodes map[int]*Node
}

type Message struct{
}

func handle(err error){
    if err != nil{
       log.Fatal(err)
    }
}

func (node *Node) Start(){
    // Open the Badger database located in the /tmp/badger directory.
    // It will be created if it doesn't exist.
    nodeString := strconv.Itoa(node.id)
    dbPath := fmt.Sprintf("/tmp/badger%d",node.id)
    db, err := badger.Open(badger.DefaultOptions(dbPath))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    initialUpdate := make(map[string][]uint8)
    newjson := JSON{
        ID: node.id,
        Name: node.name,
    }
    b, marshalErr := json.Marshal(newjson)
    handle(marshalErr)
    initialUpdate[node.name] = b
    txn := db.NewTransaction(true)
    for k,v := range initialUpdate {
      if err := txn.Set([]byte(k),[]byte(v)); err == badger.ErrTxnTooBig {
        _ = txn.Commit()
        txn = db.NewTransaction(true)
        _ = txn.Set([]byte(k),[]byte(v))
      }
    }
    _ = txn.Commit()
    //Start infinite listening
    for{
        select{
        case <- node.quitChannel:
            fmt.Printf("Node %d stopped\n",node.id)
            return
        case receivedMsg := <- node.nodeChannel:
            Msg := receivedMsg.(*Message)
            go node.HandleMessage(Msg)
        }
    }
}

func (node *Node) HandleMessage(Msg *Message){
    node.UpdateDB(msg)
}

func (node *Node) UpdateDB(msg){
    update = msg.Update
    db = node.DB
    txn := db.NewTransaction(true)
    for k,v := range update{
      if err := txn.Set([]byte(k),[]byte(v)); err == badger.ErrTxnTooBig {
        _ = txn.Commit()
        txn = db.NewTransaction(true)
        _ = txn.Set([]byte(k),[]byte(v))
      }
    }
    _ = txn.Commit()
}

func (node *Node) readDB(){
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
}

func (node *Node) updateOthers(updateMessage *Message){
    //Do Nothing
    for i:= 1; i <= len(node.Allnodes); i++ {
        if i!=node.Id{
            go func(i int){
                select{
                case node.allNodes[i].nodeChannel <- updateMessage:
                    return
                case <- time.After(1000 * time.Millisecond):
                    return
                }
            }(i)
        }
    }
}

func main() {
    numberOfNodes := 5
    fmt.Println("Initiating Nodes")
    allNodes := make(map[int]*Node) // Node ID : Pointer to Node
    for i := 1; i <= CoordinatorId; i++{
        newNode := &Node{}
        newNode.name = fmt.Sprintf("Node %d",i)
        newNode.id = i
        newNode.quitChannel = make(chan struct{})
        newNode.nodeChannel = make(chan interface{})
        newNode.nodeDBPath = fmt.Sprintf("/tmp/badger_%d",i)
        allNodes[i] = newNode
    }
    //Let the nodes know about each other
    for _,node := range allNodes{
        node.allNodes = allNodes
        go node.Start()
    }
    for _,node := range allNodes{
        go node.UpdateOthers()
    }
    // Open the Badger database located in the /tmp/badger directory.
    // It will be created if it doesn't exist.
}
