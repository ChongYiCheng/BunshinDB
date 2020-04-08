package main

import (
    "log"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestGetHandler(t *testing.T){
    client := &http.Client{
    }

    url := "http://localhost:8080/get"
    Msg = &Message{
        SenderIP:"127.0.0.1",
        SenderPort:"8081",
        MessageType: 0,
        Query: "Testing",
    }

    jsonBuffer, err := json.Marshal(msg)
    if err != nil{
        fmt.Errorf("Failed to marshal message")
    }
    
    req,err := http.NewRequest("POST",url,bytes.NewBuffer(jsonBuffer))
    req.Header.Set("Content-Type","application/json")

    res,err := client.Do(req)

    defer res.Body.Close()

    if err != nil{
        fmt.Println("Unable to reach the server.")
    } else {
        var resMsg Message
        json.NewDecoder(res.Body).Decode(&resMsg)
        fmt.Println(resMsg)
}
