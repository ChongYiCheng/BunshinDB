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
	//"./pkg/VectorClock"
	//"./pkg/Item"
	"io/ioutil"
	//"time"

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

func (client *Client) HttpClientReq(msg *Message,targetUrl string,endpoint string){
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
    defer res.Body.Close()
    fmt.Println("HTTP Client Req - Got a response")

    // always close the response-body, even if content is not required

    if err != nil {
         fmt.Println("Unable to reach the server.")
    } else{
        var resMsg Message
		json.NewDecoder(res.Body).Decode(&resMsg)
        fmt.Printf("Response Message is \n%v\n",resMsg)
        msgData := map[string]ShoppingCart.ShoppingCart{}
        if endpoint == "get" && len(msgData) > 1{
            //TODO Need to add semantic reconciliation handling case
            //Conflicting shopping cart versions, need to perform semantic reconciliation
            //and write back to coordinator


        } else{
            for k,v := range resMsg.Data{
                var shoppingCart ShoppingCart.ShoppingCart
                unMarshalErr := json.Unmarshal(v,&shoppingCart)
                if unMarshalErr != nil{
                    fmt.Errorf("Failed to unmarshal message data")
                }
                msgData[k] = shoppingCart
            }
            fmt.Printf("Data of the message is \n%v\n",msgData)
        }
    }
}

func (client *Client) semanticReconciliation(conflictedMessage Message){
    //Need to collate list of conflicted shopping carts then merge them
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
    client.HttpClientReq(httpMsg,targetUrl,"put")
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
			targetUrl := client.KnownNodeURLs[rand.Intn(len(client.KnownNodeURLs))]
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
            //rand.Intn(len(client.KnownNodeURLs)) - we remove this from the below arg for testing purpose
			targetUrl := client.KnownNodeURLs[0]
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
    //Set constants here
    //TODO need to know at least some of the members of the ring somehow
    KnownNodeUrls := []string{fmt.Sprintf("%s:8080",currentIP),fmt.Sprintf("%s:8081",currentIP),fmt.Sprintf("%s:8082",currentIP),fmt.Sprintf("%s:8083",currentIP)}

    client := &Client{currentIP,port,KnownNodeUrls}

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
