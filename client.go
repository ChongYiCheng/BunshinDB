package main

import (
    "os"
    "os/exec"
    "fmt"
    "strings"
    "encoding/json"
    "net/http"
    "bytes"
)

type Client struct{
    ip string 
    port string
    serverList []string
}


func (client *Client) httpClientReq(msg *Message,targetUrl string){
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
        fmt.Println(resMsg, "(http response message)")
        if len(resMsg.Data) == 0 {
            fmt.Println("PUT has completed")
            return 
        }
        keys := make([]string, len(resMsg.Data))
		i := 0
		for k := range resMsg.Data {
			keys[i] = k
			i++
		}
		byt := resMsg.Data[keys[0]]
		var cart ShoppingCart
		json.Unmarshal(byt, &cart)
		fmt.Println("The shopping cart is:", cart)
    }
}

func (client *Client) getCart(arrCommandStr []string) { 
    httpMsg := &Message{}
    httpMsg.SenderIP = client.ip
    httpMsg.SenderPort = client.port
    httpMsg.MessageType = "GET"
    key := arrCommandStr[3]
    httpMsg.Query = key
    targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
    client.httpClientReq(httpMsg,targetUrl)
}


func (client *Client) runCommand(commandStr string) error {
	commandStr = strings.TrimSuffix(commandStr, "\r\n")
	arrCommandStr, parseErr := parseCommandLine(commandStr)
	handle(parseErr)

    //Subcommands
    if len(arrCommandStr)>=1{
		switch arrCommandStr[0] {
		case "exit":
			os.Exit(0)
		case "httpPut":
			//Do nothing
			if len(arrCommandStr)<5{
				return fmt.Errorf("Usage of httpSend - httpSend <targetIP> <targetPort> <key> <value1> <value2>...etc")
			}
			httpMsg := &Message{}
			httpMsg.SenderIP = client.ip
			httpMsg.SenderPort = client.port
			httpMsg.MessageType = "POST"
            key := arrCommandStr[3]
            items := arrCommandStr[4:]
            shoppingCart := &ShoppingCart{Items:items}
            // rawValue := arrCommandStr[4]
            value, marshalErr := json.Marshal(shoppingCart)
            handle(marshalErr)
            data := map[string][]byte{key:value}
			httpMsg.Data = data
            fmt.Printf("httpMsg %s\r\n",httpMsg)
            targetUrl := fmt.Sprintf("%s:%s",arrCommandStr[1],arrCommandStr[2])
            client.httpClientReq(httpMsg,targetUrl)
        case "httpGet":
            if len(arrCommandStr)!=4{
                return fmt.Errorf("Usage of httpGet - httpGet <targetIP> <targetPort> <key to query>")
            }
            client.getCart(arrCommandStr)

        default:
		cmd := exec.Command(arrCommandStr[0], arrCommandStr[1:]...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		return cmd.Run()
        }
    }
    return nil
}



