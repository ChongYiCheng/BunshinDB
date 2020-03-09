package main

import (
    "encoding/json"
    "fmt"
    "os"
)

func main() {
    type ColorGroup struct {
        ID     int
        Name   string
        Colors []string
    }
    group := ColorGroup{
        ID:     1,
        Name:   "Reds",
        Colors: []string{"Crimson", "Red", "Ruby", "Maroon"},
    }
    fmt.Printf("Type of JSON before encode is %T\n",group)
    b, err := json.Marshal(group)
    if err != nil {
        fmt.Println("error:", err)
    }
	fmt.Printf("Type of json.Marshal is %T\n",b)
    os.Stdout.Write(b)
}
