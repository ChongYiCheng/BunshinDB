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

func handle(err error){
    if err != nil{
       log.Fatal(err)
    }
}


func main() {
    // Open the Badger database located in the /tmp/badger directory.
    // It will be created if it doesn't exist.
    db, err := badger.Open(badger.DefaultOptions("/tmp/badger"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    fmt.Printf("BadgerDB is type %T",db)
    // Your code here…
    updates := make(map[string][]uint8)
    for i := 1 ; i < 1000 ; i++ {
        intString := strconv.Itoa(i)
        testString := "Name " + strconv.Itoa(i)
        newjson := JSON{
            ID: i,
            Name: testString,
        }
        b, marshalErr := json.Marshal(newjson)
        handle(marshalErr)
        updates[intString]=b
    }
    txn := db.NewTransaction(true)
    for k,v := range updates {
      if err := txn.Set([]byte(k),[]byte(v)); err == badger.ErrTxnTooBig {
        _ = txn.Commit()
        txn = db.NewTransaction(true)
        _ = txn.Set([]byte(k),[]byte(v))
      }
    }
    _ = txn.Commit()

	err = db.View(func(txn *badger.Txn) error {
    for i := 1 ; i < 1000 ; i++ {
        intString := strconv.Itoa(i)
        item, err := txn.Get([]byte(intString))
        handle(err)

        var valCopy []byte
        err = item.Value(func(val []byte) error {
          // This func with val would only be called if item.Value encounters no error.

          // Accessing val here is valid.
          fmt.Printf("The answer is: %s\n", val)

          // Copying or parsing val is valid.
          valCopy = append([]byte{}, val...)

          return nil
        })
        handle(err)

        // You must copy it to use it outside item.Value(...).
        fmt.Printf("The answer is: %s\n", valCopy)
        fmt.Printf("Type of Value is %T\n",valCopy)

        // Alternatively, you could also use item.ValueCopy().
        valCopy, err = item.ValueCopy(nil)
        handle(err)
        fmt.Printf("The answer is: %s\n", valCopy)
        fmt.Printf("Type of Value is %T\n",valCopy)
    }

    return nil
    })
}
